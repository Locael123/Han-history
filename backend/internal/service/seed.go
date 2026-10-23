package service

import (
	"fmt"
	"log"
	"time"

	"gonghe/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ---------- 种子数据结构 ----------

type seedEvent struct {
	Year     int
	Month    int
	Day      int
	Title    string
	Content  string
	Location string
	Figures  string
	Tags     string
}

type seedBranch struct {
	Name        string
	Description string
	Owner       string // 空表示官修
	Official    bool
	Parent      string // 父枝名，空表示根
	ForkAt      string // 分叉点条目名
	Events      []seedEvent
}

type seedUser struct {
	Username string
	Nickname string
	Bio      string
}

type seedComment struct {
	TargetType string
	Target     string // 史枝名或条目标题
	User       string
	Content    string
	ReplyTo    string // 回复某人的评论（同一对象），可空
}

// Seed 在空库上写入一部可浏览的民国史。非空库直接跳过。
func Seed(db *gorm.DB) error {
	var userCount, branchCount int64
	if err := db.Model(&model.User{}).Count(&userCount).Error; err != nil {
		return err
	}
	if err := db.Model(&model.Branch{}).Count(&branchCount).Error; err != nil {
		return err
	}
	if userCount > 0 || branchCount > 0 {
		return nil
	}

	start := time.Now()
	hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	now := time.Now()
	users := []seedUser{
		{"shuoshuren", "说书人", "在茶馆讲野史，讲到关键处就敲醒木。"},
		{"baishanke", "白山客", "祖上闯关东，家里有一箱发黄的旧报纸。"},
		{"tingyuzhe", "听雨者", "西南联大校史爱好者，收集旧讲义与口述。"},
		{"jinling", "金陵旧客", "南京人，外婆经历过那一年。"},
		{"lingnan", "岭南生", "广州东山口长大，爱翻黄埔旧档。"},
		{"shiguan", "史馆", "官修正史编辑，只负责把大事钉在时间轴上。"},
	}
	userRows := make([]model.User, 0, len(users))
	for i, u := range users {
		userRows = append(userRows, model.User{
			Username:     u.Username,
			Nickname:     u.Nickname,
			Bio:          u.Bio,
			PasswordHash: string(hash),
			CreatedAt:    now.AddDate(0, 0, -240+i*7),
			UpdatedAt:    now.AddDate(0, 0, -240+i*7),
		})
	}
	if err := db.Create(&userRows).Error; err != nil {
		return err
	}
	userByName := make(map[string]model.User, len(userRows))
	userByNickname := make(map[string]model.User, len(userRows))
	for _, u := range userRows {
		userByName[u.Username] = u
		userByNickname[u.Nickname] = u
	}
	// 种子内容里用户名和雅号混着写，两个索引都查。
	findUser := func(key string) (model.User, bool) {
		if u, ok := userByName[key]; ok {
			return u, true
		}
		u, ok := userByNickname[key]
		return u, ok
	}
	historianID := userByName["shiguan"].ID

	// ---------- 正史主干 ----------

	trunk := model.Branch{
		Name:        "正史·共和本纪",
		Description: "自辛亥翌年民国肇建，至己丑鼎革。三十八年，五易国都，三度内战，一次外侮。本纪只记大事，不记人心。",
		OwnerID:     0,
		Official:    true,
		CreatedAt:   now.AddDate(0, 0, -180),
		UpdatedAt:   now.AddDate(0, 0, -180),
	}
	if err := db.Create(&trunk).Error; err != nil {
		return err
	}
	trunkEvents, err := insertEvents(db, trunk.ID, historianID, trunkSeed, now.AddDate(0, 0, -179))
	if err != nil {
		return err
	}

	branchByName := map[string]model.Branch{trunk.Name: trunk}
	eventByTitle := map[string]model.Event{}
	for _, e := range trunkEvents {
		eventByTitle[e.Title] = e
	}

	// ---------- 其他史枝 ----------

	for bi, def := range branchSeeds {
		parent, ok := branchByName[def.Parent]
		if !ok {
			return fmt.Errorf("种子数据：父枝 %q 不存在", def.Parent)
		}
		forkEvent, ok := eventByTitle[def.ForkAt]
		if !ok {
			return fmt.Errorf("种子数据：分叉点 %q 不存在", def.ForkAt)
		}
		inherited, err := countThroughEvent(db, parent.ID, forkEvent.ID)
		if err != nil {
			return err
		}

		ownerID := uint64(0)
		if !def.Official {
			u, ok := findUser(def.Owner)
			if !ok {
				return fmt.Errorf("种子数据：用户 %q 不存在", def.Owner)
			}
			ownerID = u.ID
		}

		createdAt := now.AddDate(0, 0, -120+bi*12)
		b := model.Branch{
			Name:           def.Name,
			Description:    def.Description,
			OwnerID:        ownerID,
			Official:       def.Official,
			ParentID:       parent.ID,
			ForkEventID:    forkEvent.ID,
			ForkYear:       forkEvent.Year,
			InheritedCount: inherited,
			CreatedAt:      createdAt,
			UpdatedAt:      createdAt,
		}
		if err := db.Create(&b).Error; err != nil {
			return err
		}
		authorID := historianID
		if ownerID != 0 {
			authorID = ownerID
		}
		rows, err := insertEvents(db, b.ID, authorID, def.Events, createdAt)
		if err != nil {
			return err
		}
		for _, e := range rows {
			eventByTitle[e.Title] = e
		}
		branchByName[b.Name] = b
		if err := refreshBranchStats(db, b.ID); err != nil {
			return err
		}
	}
	if err := refreshBranchStats(db, trunk.ID); err != nil {
		return err
	}

	// ---------- 点赞 ----------

	type likeSpec struct {
		User       string
		TargetType string
		Target     string
	}
	likeSpecs := []likeSpec{
		{"shuoshuren", model.TargetBranch, "正史·共和本纪"},
		{"baishanke", model.TargetBranch, "正史·共和本纪"},
		{"tingyuzhe", model.TargetBranch, "正史·共和本纪"},
		{"jinling", model.TargetBranch, "正史·共和本纪"},
		{"lingnan", model.TargetBranch, "正史·共和本纪"},
		{"shiguan", model.TargetBranch, "正史·共和本纪"},
		{"shuoshuren", model.TargetBranch, "野史·沪上旧梦"},
		{"tingyuzhe", model.TargetBranch, "野史·沪上旧梦"},
		{"jinling", model.TargetBranch, "野史·沪上旧梦"},
		{"baishanke", model.TargetBranch, "野史·关外三十年纪事"},
		{"shuoshuren", model.TargetBranch, "野史·关外三十年纪事"},
		{"tingyuzhe", model.TargetBranch, "野史·联大八年"},
		{"lingnan", model.TargetBranch, "野史·联大八年"},
		{"jinling", model.TargetBranch, "野史·金陵残照"},
		{"tingyuzhe", model.TargetBranch, "野史·金陵残照"},
		{"lingnan", model.TargetBranch, "野史·岭南旧事"},
		{"baishanke", model.TargetBranch, "野史·岭南旧事"},
		{"shiguan", model.TargetBranch, "别史·新潮"},
		{"shiguan", model.TargetBranch, "别史·实业志"},

		{"shuoshuren", model.TargetEvent, "清帝退位"},
		{"tingyuzhe", model.TargetEvent, "清帝退位"},
		{"jinling", model.TargetEvent, "清帝退位"},
		{"shuoshuren", model.TargetEvent, "五四运动"},
		{"baishanke", model.TargetEvent, "五四运动"},
		{"tingyuzhe", model.TargetEvent, "五四运动"},
		{"jinling", model.TargetEvent, "五四运动"},
		{"lingnan", model.TargetEvent, "五四运动"},
		{"shuoshuren", model.TargetEvent, "四一二"},
		{"baishanke", model.TargetEvent, "四一二"},
		{"jinling", model.TargetEvent, "四一二"},
		{"shuoshuren", model.TargetEvent, "南京陷落"},
		{"baishanke", model.TargetEvent, "南京陷落"},
		{"tingyuzhe", model.TargetEvent, "南京陷落"},
		{"jinling", model.TargetEvent, "南京陷落"},
		{"lingnan", model.TargetEvent, "开国大典"},
		{"shuoshuren", model.TargetEvent, "开国大典"},
		{"tingyuzhe", model.TargetEvent, "开国大典"},
		{"baishanke", model.TargetEvent, "开国大典"},
		{"shuoshuren", model.TargetEvent, "百乐门开张"},
		{"tingyuzhe", model.TargetEvent, "百乐门开张"},
		{"jinling", model.TargetEvent, "阮玲玉之死"},
		{"lingnan", model.TargetEvent, "阮玲玉之死"},
		{"shuoshuren", model.TargetEvent, "阮玲玉之死"},
		{"tingyuzhe", model.TargetEvent, "西南联大成立"},
		{"lingnan", model.TargetEvent, "西南联大成立"},
		{"baishanke", model.TargetEvent, "西南联大成立"},
		{"baishanke", model.TargetEvent, "伪满洲国成立"},
		{"shuoshuren", model.TargetEvent, "伪满洲国成立"},
		{"jinling", model.TargetEvent, "南京受降"},
		{"tingyuzhe", model.TargetEvent, "南京受降"},
		{"shuoshuren", model.TargetEvent, "南京受降"},
		{"lingnan", model.TargetEvent, "黄埔军校开学"},
		{"baishanke", model.TargetEvent, "黄埔军校开学"},
	}
	seenLike := map[string]bool{}
	for _, spec := range likeSpecs {
		u, ok := findUser(spec.User)
		if !ok {
			continue
		}
		var targetID uint64
		switch spec.TargetType {
		case model.TargetBranch:
			b, ok := branchByName[spec.Target]
			if !ok {
				continue
			}
			targetID = b.ID
		case model.TargetEvent:
			e, ok := eventByTitle[spec.Target]
			if !ok {
				continue
			}
			targetID = e.ID
		}
		key := fmt.Sprintf("%d|%s|%d", u.ID, spec.TargetType, targetID)
		if seenLike[key] {
			continue
		}
		seenLike[key] = true
		if err := db.Create(&model.Like{
			UserID:     u.ID,
			TargetType: spec.TargetType,
			TargetID:   targetID,
			CreatedAt:  now.AddDate(0, 0, -3),
		}).Error; err != nil {
			return err
		}
	}

	// ---------- 评论 ----------

	commentSeeds := []seedComment{
		{model.TargetBranch, "正史·共和本纪", "说书人", "正史是骨架，野史才是血肉。", ""},
		{model.TargetBranch, "正史·共和本纪", "白山客", "关外那十四年，在本纪里只占两行，实在太轻。", ""},
		{model.TargetBranch, "野史·沪上旧梦", "听雨者", "百乐门那一段写得像旧画报，连香水味都出来了。", ""},
		{model.TargetBranch, "野史·沪上旧梦", "金陵旧客", "阮玲玉出殡那年，我祖母就在上海，说人山人海。", ""},
		{model.TargetBranch, "野史·关外三十年纪事", "说书人", "十四年，一句话带过，太轻了。", ""},
		{model.TargetBranch, "野史·联大八年", "岭南生", "茅屋铁皮，弦歌不辍，这八个字就够了。", ""},
		{model.TargetEvent, "四一二", "白山客", "这一页太沉，翻过去手都是抖的。", ""},
		{model.TargetEvent, "百乐门开张", "说书人", "开张那晚，据说全上海的汽车都堵在静安寺路。", ""},
		{model.TargetEvent, "百乐门开张", "听雨者", "堵车我信，老照片里车牌一直排到愚园路。", "说书人"},
		{model.TargetEvent, "西南联大成立", "岭南生", "铁皮屋顶下听课，雨声大过先生的声音。", ""},
		{model.TargetEvent, "南京受降", "金陵旧客", "外公当年在中央军校外围维持秩序，回来只说了一句：结束了。", ""},
		{model.TargetEvent, "开国大典", "说书人", "一个时代落幕，另一个时代开始。", ""},
		{model.TargetEvent, "日本宣布投降", "tingyuzhe", "重庆的鞭炮声，隔了八十多年还听得见。", ""},
	}
	commentKey := map[string]uint64{}
	commentUserKey := func(targetType, target, username string) string {
		return targetType + "|" + target + "|" + username
	}
	for i, c := range commentSeeds {
		u, ok := findUser(c.User)
		if !ok {
			continue
		}
		var targetID uint64
		switch c.TargetType {
		case model.TargetBranch:
			b, ok := branchByName[c.Target]
			if !ok {
				continue
			}
			targetID = b.ID
		case model.TargetEvent:
			e, ok := eventByTitle[c.Target]
			if !ok {
				continue
			}
			targetID = e.ID
		}
		replyID := uint64(0)
		replyName := ""
		if c.ReplyTo != "" {
			if id, ok := commentKey[commentUserKey(c.TargetType, c.Target, c.ReplyTo)]; ok {
				replyID = id
				if ru, ok := findUser(c.ReplyTo); ok {
					replyName = ru.Nickname
				}
			}
		}
		row := model.Comment{
			TargetType:  c.TargetType,
			TargetID:    targetID,
			UserID:      u.ID,
			ReplyToID:   replyID,
			ReplyToName: replyName,
			Content:     c.Content,
			CreatedAt:   now.Add(-time.Duration(i*7+2) * time.Hour),
		}
		if err := db.Create(&row).Error; err != nil {
			return err
		}
		commentKey[commentUserKey(c.TargetType, c.Target, c.User)] = row.ID
	}

	if err := recalcCounters(db); err != nil {
		return err
	}
	log.Printf("[seed] 已写入 %d 位史官、%d 条史枝、%d 条实录、%d 条评论（耗时 %s）",
		len(users), len(branchByName), len(eventByTitle), len(commentSeeds), time.Since(start).Round(time.Millisecond))
	return nil
}

// insertEvents 按顺序插入条目，返回带 ID 的行。
func insertEvents(db *gorm.DB, branchID, authorID uint64, seeds []seedEvent, base time.Time) ([]model.Event, error) {
	out := make([]model.Event, 0, len(seeds))
	for i, s := range seeds {
		createdAt := base.Add(time.Duration(i*9) * time.Hour)
		e := model.Event{
			BranchID:  branchID,
			UserID:    authorID,
			Title:     s.Title,
			Content:   s.Content,
			Year:      s.Year,
			Month:     s.Month,
			Day:       s.Day,
			Location:  s.Location,
			Figures:   s.Figures,
			Tags:      s.Tags,
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		}
		if err := db.Create(&e).Error; err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

// countThroughEvent 计算父枝（含更早祖先）到某个条目为止的条目数。
func countThroughEvent(db *gorm.DB, branchID, eventID uint64) (int, error) {
	var rows []model.Event
	if err := db.Where("branch_id = ?", branchID).Order("id ASC").Find(&rows).Error; err != nil {
		return 0, err
	}
	for i, e := range rows {
		if e.ID == eventID {
			return i + 1, nil
		}
	}
	return 0, fmt.Errorf("分叉点 %d 不属于枝 %d", eventID, branchID)
}

// recalcCounters 在所有种子写完后统一校准反规范化计数。
// 比边写边加减更不容易出错，种子只跑一次，成本可以忽略。
func recalcCounters(db *gorm.DB) error {
	stmts := []string{
		`UPDATE branches SET fork_count = (SELECT COUNT(*) FROM branches c WHERE c.parent_id = branches.id)`,
		`UPDATE branches SET like_count = (SELECT COUNT(*) FROM likes l WHERE l.target_type = 'branch' AND l.target_id = branches.id)`,
		`UPDATE branches SET comment_count = (SELECT COUNT(*) FROM comments c WHERE c.target_type = 'branch' AND c.target_id = branches.id)`,
		`UPDATE events SET like_count = (SELECT COUNT(*) FROM likes l WHERE l.target_type = 'event' AND l.target_id = events.id)`,
		`UPDATE events SET comment_count = (SELECT COUNT(*) FROM comments c WHERE c.target_type = 'event' AND c.target_id = events.id)`,
		`UPDATE comments SET like_count = (SELECT COUNT(*) FROM likes l WHERE l.target_type = 'comment' AND l.target_id = comments.id)`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

// ---------- 内容 ----------

var trunkSeed = []seedEvent{
	{1912, 1, 1, "南京临时政府成立", "孙中山自海外归国，十七省代表聚于南京，选举临时大总统。翌日，中华民国临时政府成立，改用阳历，以五色旗为国旗。数千年帝制，至此暂告一段落。", "南京", "孙中山，黄兴，宋教仁", "共和，建国"},
	{1912, 2, 12, "清帝退位", "隆裕太后以宣统皇帝名义颁退位诏书，谓「将统治权公诸全国，定为共和立宪国体」。诏书一出，南北议和告成。北京城头的黄龙旗缓缓落下，有人剪辫，有人痛哭。", "北京", "溥仪，袁世凯，隆裕太后", "共和，鼎革"},
	{1912, 3, 10, "袁世凯就任临时大总统", "袁氏于北京就任临时大总统，临时政府迁都北京。南方革命党人多有不满，然兵饷两绌，只得退让。自此，北洋之局渐成。", "北京", "袁世凯，黎元洪", "共和，北洋"},
	{1913, 7, 12, "二次革命", "李烈钧于湖口起兵讨袁，黄兴响应于南京。然仓促举事，不足两月即告失败。孙中山再走日本，国民党被解散。民国初年的议会政治，至此元气大伤。", "江西，南京", "李烈钧，孙中山，黄兴", "兵事，共和"},
	{1915, 1, 18, "二十一条", "日本以支持帝制为饵，提出二十一条。袁政府虽经删改，仍签「民四条约」。消息传出，举国哗然，抵制日货之声遍及通商口岸。五月九日，后被定为国耻纪念。", "北京，东京", "袁世凯，日置益", "外交，国耻"},
	{1915, 9, 15, "《青年杂志》创刊", "陈独秀于上海创办《青年杂志》，倡「自主的而非奴隶的，进步的而非保守的」。次年更名《新青年》。一本薄薄的刊物，日后竟搅动整个中国的思想界。", "上海", "陈独秀", "文化，新潮"},
	{1915, 12, 12, "袁世凯称帝", "参政院拥戴袁世凯为皇帝，改国号「中华帝国」，年号洪宪。蔡锷、唐继尧等于云南通电独立，护国战争起。帝制仅八十三日，即告收场。", "北京", "袁世凯，杨度，蔡锷", "帝制，兵事"},
	{1916, 6, 6, "袁世凯病逝", "袁世凯病逝于中南海居仁堂。黎元洪继任总统，段祺瑞任国务总理，南北重开和议。然北洋群龙无首，各系军阀自此割据，政令不出都门。", "北京", "袁世凯，黎元洪，段祺瑞", "北洋，政局"},
	{1917, 7, 1, "张勋复辟", "张勋率辫子军入京，拥溥仪复辟，改民国六年为宣统九年。段祺瑞马厂誓师，讨逆军十二日即攻入北京。复辟不过十二日，却成了一个时代的注脚。", "北京", "张勋，溥仪，段祺瑞", "复辟，共和"},
	{1917, 8, 25, "护法运动", "孙中山率海军南下广州，召开国会非常会议，组护法军政府。南北分裂之势，自此延续十余年。", "广州", "孙中山，程璧光", "南北，兵事"},
	{1919, 5, 4, "五四运动", "巴黎和会决定将德国在山东权益让与日本，消息传回，北京学生三千余人齐集天安门，火烧赵家楼。罢课、罢市、罢工席卷全国。新文化运动自此由书斋走入街头。", "北京", "蔡元培，陈独秀，学生", "五四，新潮"},
	{1921, 7, 23, "中共成立", "各地代表十三人聚于上海法租界，后转嘉兴南湖一叶画舫。中国共产党第一次全国代表大会通过党纲，选举陈独秀为书记。当时无人料到，这将是改变中国命运的开端。", "上海，嘉兴", "毛泽东，李达，董必武", "政党，赤潮"},
	{1924, 1, 20, "国民党一大", "中国国民党第一次全国代表大会于广州召开，确立联俄、联共、扶助农工三大政策，重新解释三民主义。国共第一次合作由此开始，黄埔军校随后创办。", "广州", "孙中山，廖仲恺，李大钊", "政党，合作"},
	{1925, 3, 12, "孙中山逝世", "孙中山病逝于北京铁狮子胡同行辕，遗言「革命尚未成功，同志仍须努力」。灵柩暂厝香山碧云寺，举国哀悼。此后国民党内，再无人能服众。", "北京", "孙中山，汪精卫", "共和，伟人"},
	{1926, 7, 9, "国民革命军北伐", "国民革命军于广州誓师北伐，分三路入湘、赣、闽。不到一年，克武汉、下南京、定上海。北洋军阀的统治，在枪炮声中土崩瓦解。", "广州", "蒋介石，李宗仁，唐生智", "北伐，兵事"},
	{1927, 4, 12, "四一二", "上海总工会工人纠察队被缴械，大规模搜捕与处决随即展开。国共第一次合作破裂，宁汉分裂，白色恐怖蔓延。此后二十余年，两党兵戎相见。", "上海", "蒋介石，工人纠察队", "清党，政党"},
	{1927, 4, 18, "南京国民政府成立", "南京国民政府宣告成立，与武汉国民政府对立。七月，宁汉合流；次年，东北易帜，全国名义上重归统一。黄金十年，自此开端。", "南京", "蒋介石，胡汉民", "建政，共和"},
	{1928, 12, 29, "东北易帜", "张学良宣布东北改悬青天白日旗，服从南京国民政府。至此，中国在名义上完成统一。距清帝退位，已十七年。", "沈阳", "张学良，蒋介石", "统一，东北"},
	{1931, 9, 18, "九一八事变", "日本关东军炸毁南满铁路柳条湖段，反诬中国军队，炮击北大营。东北军奉命不抵抗，四月之内，东三省尽失。白山黑水，从此沦陷十四年。", "沈阳", "关东军，张学良", "国难，兵事"},
	{1936, 12, 12, "西安事变", "张学良、杨虎城于华清池扣留蒋介石，通电要求停止内战、一致抗日。经周恩来等多方斡旋，蒋氏允诺联共抗日，事变得以和平解决。十年内战，至此基本结束。", "西安", "张学良，杨虎城，蒋介石", "转折，兵事"},
	{1937, 7, 7, "卢沟桥事变", "日军于卢沟桥畔挑衅，二十九军奋起还击。蒋介石在庐山发表谈话：「地无分南北，人无分老幼，皆有守土抗战之责。」全面抗战，自此爆发。", "北平", "二十九军，佟麟阁，赵登禹", "抗战，兵事"},
	{1937, 12, 13, "南京陷落", "南京城破，日军入城后展开长达六周的大屠杀，遇难者三十万以上。国民政府迁都重庆，抗战进入最艰难的阶段。", "南京", "日军，唐生智", "国难，抗战"},
	{1945, 8, 15, "日本宣布投降", "日本天皇广播《终战诏书》，宣布无条件投降。重庆街头鞭炮彻夜，万人空巷。八年浴血，终于等到这一天。九月九日，南京受降仪式举行。", "重庆，南京", "裕仁，蒋介石", "胜利，抗战"},
	{1945, 10, 10, "双十协定", "国共双方于重庆签署《政府与中共代表会谈纪要》，即双十协定，确认和平建国方针。然而纸墨未干，摩擦已起。", "重庆", "毛泽东，蒋介石", "和谈，政党"},
	{1946, 6, 26, "全面内战爆发", "国民党军队大举进攻中原解放区，全面内战爆发。此后三年，辽沈、淮海、平津三大战役，江山易手。", "中原", "国民党，解放军", "内战，兵事"},
	{1949, 10, 1, "开国大典", "毛泽东于天安门城楼宣告中华人民共和国中央人民政府成立。同日，国民政府自广州迁往重庆，年底再迁台北。一个时代落幕，另一个时代开始。", "北京", "毛泽东", "鼎革，共和"},
}

var branchSeeds = []seedBranch{
	{
		Name:        "别史·新潮",
		Description: "从《青年杂志》创刊处另起一脉，专记文章、刊物与思想。正史写谁掌权，别史写谁改变了语言。",
		Official:    true,
		Parent:      "正史·共和本纪",
		ForkAt:      "《青年杂志》创刊",
		Events: []seedEvent{
			{1917, 1, 1, "《文学改良刍议》", "胡适于《新青年》发表《文学改良刍议》，主张「不摹仿古人、不避俗字俗语」等八事。陈独秀继以《文学革命论》响应。白话文自此渐成气候。", "北京", "胡适，陈独秀", "文学，新潮"},
			{1918, 5, 15, "《狂人日记》发表", "鲁迅以「鲁迅」为笔名，在《新青年》发表中国第一篇现代白话小说《狂人日记》。字缝里看出两个字：吃人。", "北京", "鲁迅，钱玄同", "小说，新潮"},
			{1920, 1, 12, "国语运动", "教育部令国民学校一二年级改国文为语体文。白话文正式进入课堂，千年的「之乎者也」让位于「的了吗呢」。", "北京", "教育部，胡适", "教育，国语"},
			{1923, 2, 14, "科玄论战", "张君劢在清华演讲「人生观」，主张科学不能解决人生问题；丁文江撰文驳之。科学与玄学之争震动学界，也划开了此后百年的思想分野。", "北京", "张君劢，丁文江，梁启超", "思想，论战"},
		},
	},
	{
		Name:        "别史·实业志",
		Description: "自民国肇建分出的经济一线：欧战的短暂春天、交易所的疯狂、法币改革的豪赌。",
		Official:    true,
		Parent:      "正史·共和本纪",
		ForkAt:      "袁世凯就任临时大总统",
		Events: []seedEvent{
			{1914, 8, 1, "欧战与民族工业", "欧战起，列强无暇东顾，中国民族纺织、面粉业迎来短暂春天。荣氏兄弟的纱厂昼夜不停，张謇在南通扩园办学。黄金十年之前，先有这四年。", "上海，南通", "荣宗敬，张謇，穆藕初", "实业，经济"},
			{1920, 7, 1, "上海证券物品交易所开业", "上海证券物品交易所开业，纱花、粮油、金银一并交易。投机之风大炽，蒋介石亦曾是其中经纪，亏空之后才南下广州。", "上海", "虞洽卿，蒋介石", "金融，投机"},
			{1935, 11, 4, "法币改革", "国民政府宣布废止银本位，以中央、中国、交通三行纸币为法币。统一币制，亦埋下通胀之根。", "南京", "孔祥熙，宋子文", "币制，经济"},
		},
	},
	{
		Name:        "野史·沪上旧梦",
		Description: "正史写到四一二便翻页，这里偏要留在上海。霓虹、舞池、电影明星与弄堂口的风，都是史。",
		Owner:       "shuoshuren",
		Parent:      "正史·共和本纪",
		ForkAt:      "四一二",
		Events: []seedEvent{
			{1927, 7, 7, "上海特别市成立", "上海特别市成立，华界与租界并立。十里洋场，霓虹初上，电车叮当穿过法租界的梧桐。一个城市，三种法律，五种货币。", "上海", "黄郛", "市政，沪上"},
			{1930, 3, 1, "百乐门开张", "静安寺路口，百乐门舞厅开张。玻璃舞池、爵士乐队、白俄舞女，号称「远东第一乐府」。门票一元，够一个纱厂女工做十天。", "上海", "顾联承", "风月，沪上"},
			{1932, 1, 28, "闸北的三十三天", "日军进攻闸北，十九路军奋起抵抗，血战三十三日。商务印书馆东方图书馆被焚，数十万册藏书化为灰烬。此后上海成为非武装区，孤岛时代开始。", "上海", "十九路军，蔡廷锴，蒋光鼐", "淞沪，兵事"},
			{1935, 3, 8, "阮玲玉之死", "影星阮玲玉服安眠药自尽，年二十五。出殡之日，十万市民相送，遗书只有四个字：人言可畏。", "上海", "阮玲玉，唐季珊", "影事，风月"},
			{1937, 8, 13, "八一三淞沪会战", "日军进攻上海，中国军队投入七十余万，血战三月。四行仓库的枪声，成为孤岛最后的回响。十一月，上海华界尽失，租界成为孤岛。", "上海", "张治中，谢晋元", "淞沪，抗战"},
		},
	},
	{
		Name:        "野史·关外三十年纪事",
		Description: "九一八之后，关外十四年。正史只有一行「东北沦陷」，这里把那一行拆开。",
		Owner:       "baishanke",
		Parent:      "正史·共和本纪",
		ForkAt:      "九一八事变",
		Events: []seedEvent{
			{1932, 3, 1, "伪满洲国成立", "日本扶植溥仪于长春成立「满洲国」，改元大同。国际联盟派李顿调查团，最终只换来一纸空文。国联的软弱，让东京看得清清楚楚。", "长春", "溥仪，板垣征四郎", "伪满，关外"},
			{1934, 3, 1, "溥仪称帝", "溥仪于长春称帝，改元康德。关东军司令官立于侧，所谓皇帝，不过笼中之鸟。", "长春", "溥仪，关东军", "伪满，关外"},
			{1938, 7, 1, "张鼓峰事件", "日苏两军于张鼓峰交火，伤亡数千。日本自此北进受挫，转向南进。关东军的刀，第一次撞上钢铁。", "吉林", "日军，苏联红军", "边事，关外"},
			{1945, 8, 9, "苏军出兵东北", "苏联对日宣战，百万红军越过大兴安岭。伪满覆灭，溥仪在通化被俘。关外十四年，至此终结。", "满洲里，通化", "苏联红军，溥仪", "终局，关外"},
		},
	},
	{
		Name:        "野史·联大八年",
		Description: "从卢沟桥分叉。炮火里上课，茅屋里做学问，这一枝只记读书人。",
		Owner:       "tingyuzhe",
		Parent:      "正史·共和本纪",
		ForkAt:      "卢沟桥事变",
		Events: []seedEvent{
			{1937, 9, 10, "长沙临时大学", "北大、清华、南开三校南迁长沙，组临时大学。弦歌不辍，于炮火中开学。校舍不够，便在圣经学院与韭菜园租屋上课。", "长沙", "梅贻琦，蒋梦麟，张伯苓", "教育，南迁"},
			{1938, 4, 2, "西南联大成立", "临时大学迁昆明，改称国立西南联合大学。茅屋为舍，铁皮为顶，却走出了两位诺贝尔奖得主与百余位院士。", "昆明", "三校师生", "教育，联大"},
			{1942, 5, 1, "远征军入缅", "中国远征军入缅作战，戴安澜殉国于缅北。滇缅公路，成为抗战最后的输血管。联大学生投笔从戎者，不在少数。", "滇缅", "戴安澜，孙立人", "远征，兵事"},
			{1946, 5, 4, "联大复员", "西南联大结束，三校北返。临行立碑：「八年之久，合作无间。」碑立于昆明师范学院，至今犹在。", "昆明", "三校", "教育，复员"},
		},
	},
	{
		Name:        "野史·金陵残照",
		Description: "南京城破之后另起。难民区、伪府、受降，一座城的三年零九个月。",
		Owner:       "jinling",
		Parent:      "正史·共和本纪",
		ForkAt:      "南京陷落",
		Events: []seedEvent{
			{1938, 1, 1, "金陵女子文理学院难民营", "金陵女子文理学院设难民收容所，魏特琳女士守门拒兵，护妇孺万余人。她后来在日记里写：「今天，又有多少个姑娘被带走。」", "南京", "魏特琳，难民", "南京，难民"},
			{1940, 3, 30, "汪伪政权成立", "汪精卫于南京成立「国民政府」，仍用青天白日旗，另加三角黄布。渝宁两府，各称正统。", "南京", "汪精卫，周佛海", "伪府，南京"},
			{1945, 9, 9, "南京受降", "中国战区受降仪式于南京中央军校大礼堂举行，冈村宁次呈交佩刀。八年抗战，于此画上句号。", "南京", "何应钦，冈村宁次", "受降，南京"},
		},
	},
	{
		Name:        "野史·岭南旧事",
		Description: "从国民党一大分叉。黄埔、省港、中山舰，南方的枪与口号。",
		Owner:       "lingnan",
		Parent:      "正史·共和本纪",
		ForkAt:      "国民党一大",
		Events: []seedEvent{
			{1924, 6, 16, "黄埔军校开学", "黄埔军校开学，孙中山亲题校训「亲爱精诚」。此后二十年，国共两党的将星，多出于此门。", "广州", "孙中山，蒋介石，周恩来", "军校，岭南"},
			{1925, 6, 19, "省港大罢工", "省港工人大罢工，历时十六个月，香港一度成为「死港」。罢工工人封锁海口，广州国民政府由此站稳。", "广州，香港", "苏兆征，邓中夏", "工运，岭南"},
			{1926, 3, 20, "中山舰事件", "蒋介石以中山舰调动为由，宣布戒严，逮捕共产党员。国共裂痕，初见端倪。", "广州", "蒋介石，李之龙", "政争，岭南"},
		},
	},
}
