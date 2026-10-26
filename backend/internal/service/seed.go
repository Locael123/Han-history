package service

import (
	crand "crypto/rand"
	"encoding/hex"
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

// Seed 在空库上写入一部可浏览的共和国史。非空库直接跳过。
//
// withDemo 为 false 时仍写入正史主干与史枝，但不写示例点赞、评论，
// 并给演示账号设置随机口令——公开部署可借此避免 123456 账号。
func Seed(db *gorm.DB, withDemo bool) error {
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
	password := "123456"
	if !withDemo {
		buf := make([]byte, 18)
		if _, err := crand.Read(buf); err != nil {
			return err
		}
		password = hex.EncodeToString(buf)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	now := time.Now()
	users := []seedUser{
		{"shuoshuren", "说书人", "在茶馆讲近几十年的旧事，讲到关键处就敲醒木。"},
		{"baishanke", "白山客", "祖上闯关东，家里有一箱发黄的旧报纸。"},
		{"tingyuzhe", "听雨者", "三线子弟，收集旧讲义与山沟里的口述。"},
		{"jinling", "金陵旧客", "南京人，外婆在长江大桥通车那年进城。"},
		{"lingnan", "岭南生", "广州东山口长大，爱翻南粤旧档。"},
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
		Description: "自己丑开国，至今七十七年。本纪只记大事，不记人心；只钉时间，不作评断。",
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

	commentTotal := 0
	if withDemo {

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
			{"baishanke", model.TargetBranch, "野史·关外纪事"},
			{"shuoshuren", model.TargetBranch, "野史·关外纪事"},
			{"tingyuzhe", model.TargetBranch, "野史·三线纪事"},
			{"lingnan", model.TargetBranch, "野史·三线纪事"},
			{"jinling", model.TargetBranch, "野史·金陵旧事"},
			{"tingyuzhe", model.TargetBranch, "野史·金陵旧事"},
			{"lingnan", model.TargetBranch, "野史·南粤潮声"},
			{"baishanke", model.TargetBranch, "野史·南粤潮声"},
			{"shiguan", model.TargetBranch, "别史·新潮"},
			{"shiguan", model.TargetBranch, "别史·实业志"},

			{"shuoshuren", model.TargetEvent, "开国大典"},
			{"tingyuzhe", model.TargetEvent, "开国大典"},
			{"jinling", model.TargetEvent, "开国大典"},
			{"shuoshuren", model.TargetEvent, "志愿军入朝"},
			{"baishanke", model.TargetEvent, "志愿军入朝"},
			{"lingnan", model.TargetEvent, "志愿军入朝"},
			{"jinling", model.TargetEvent, "唐山地震"},
			{"baishanke", model.TargetEvent, "唐山地震"},
			{"tingyuzhe", model.TargetEvent, "恢复高考"},
			{"shuoshuren", model.TargetEvent, "恢复高考"},
			{"lingnan", model.TargetEvent, "恢复高考"},
			{"shuoshuren", model.TargetEvent, "邓小平南巡"},
			{"baishanke", model.TargetEvent, "邓小平南巡"},
			{"jinling", model.TargetEvent, "邓小平南巡"},
			{"lingnan", model.TargetEvent, "香港回归"},
			{"jinling", model.TargetEvent, "香港回归"},
			{"shuoshuren", model.TargetEvent, "神舟五号"},
			{"tingyuzhe", model.TargetEvent, "神舟五号"},
			{"baishanke", model.TargetEvent, "汶川地震"},
			{"jinling", model.TargetEvent, "汶川地震"},
			{"lingnan", model.TargetEvent, "汶川地震"},
			{"shuoshuren", model.TargetEvent, "武汉封城"},
			{"tingyuzhe", model.TargetEvent, "武汉封城"},
			{"jinling", model.TargetEvent, "武汉封城"},
			{"shuoshuren", model.TargetEvent, "北京冬奥"},
			{"lingnan", model.TargetEvent, "北京冬奥"},
			{"baishanke", model.TargetEvent, "九三阅兵"},
			{"tingyuzhe", model.TargetEvent, "九三阅兵"},
			{"jinling", model.TargetEvent, "台风蔷琵"},
			{"shuoshuren", model.TargetEvent, "台风蔷琵"},
			{"lingnan", model.TargetEvent, "巴青地震"},
			{"tingyuzhe", model.TargetEvent, "巴青地震"},
			{"baishanke", model.TargetEvent, "渝昆高铁铺轨贯通"},
			{"shuoshuren", model.TargetEvent, "渝昆高铁铺轨贯通"},
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
			{model.TargetBranch, "正史·共和本纪", "白山客", "关外那几十年，在本纪里只占几行，实在太轻。", ""},
			{model.TargetBranch, "野史·沪上旧梦", "听雨者", "认购证那一段写得像旧画报，连汗味都出来了。", ""},
			{model.TargetBranch, "野史·沪上旧梦", "金陵旧客", "九三年我出差到上海，南京路上人人都在谈股票。", ""},
			{model.TargetBranch, "野史·关外纪事", "说书人", "几十年，一句话带过，太轻了。", ""},
			{model.TargetBranch, "野史·三线纪事", "岭南生", "山沟里的工厂，地图上找不到，一代人却都记得。", ""},
			{model.TargetEvent, "开国大典", "说书人", "一个时代落幕，另一个时代开始。", ""},
			{model.TargetEvent, "恢复高考", "听雨者", "我父亲说，那张准考证他留了一辈子。", ""},
			{model.TargetEvent, "恢复高考", "白山客", "考场里坐着的，有十六岁的，也有三十岁的。", "听雨者"},
			{model.TargetEvent, "唐山地震", "baishanke", "那年夏天，北方的风里都是灰。", ""},
			{model.TargetEvent, "邓小平南巡", "lingnan", "春风吹过深圳湾，很多人的人生从那年改道。", ""},
			{model.TargetEvent, "香港回归", "岭南生", "罗湖桥头那晚，雨很大，人很多。", ""},
			{model.TargetEvent, "汶川地震", "jinling", "那天下午，全中国的电视都停在同一个画面。", ""},
			{model.TargetEvent, "武汉封城", "tingyuzhe", "后来我们才知道，那是一个时代的断点。", ""},
			{model.TargetEvent, "北京冬奥", "shuoshuren", "冰场上的光，和鸟巢上空的雪，都是新的。", ""},
			{model.TargetEvent, "九三阅兵", "baishanke", "老兵方队过来时，台下很多人站起来。", ""},
			{model.TargetEvent, "台风蔷琵", "jinling", "闽浙沿海的朋友说，那一夜雨声像鼓。", ""},
			{model.TargetEvent, "渝昆高铁铺轨贯通", "lingnan", "昆明到重庆两个半小时，山还是那座山。", ""},
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
		commentTotal = len(commentSeeds)
	}

	if err := recalcCounters(db); err != nil {
		return err
	}
	log.Printf("[seed] 已写入 %d 位史官、%d 条史枝、%d 条实录、%d 条评论（耗时 %s）",
		len(users), len(branchByName), len(eventByTitle), commentTotal, time.Since(start).Round(time.Millisecond))
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
	{1949, 10, 1, "开国大典", "毛泽东于天安门城楼宣告中华人民共和国中央人民政府成立。同日，国民政府自广州迁往重庆，年底再迁台北。一个时代落幕，另一个时代开始。", "北京", "毛泽东", "鼎革，共和"},
	{1950, 6, 25, "朝鲜战争爆发", "朝鲜半岛战事骤起，美国第七舰队驶入台湾海峡。东北边防一夜之间绷紧，新生的共和国面临第一场外部考验。", "朝鲜半岛，台北", "杜鲁门，金日成", "战事，外交"},
	{1950, 10, 19, "志愿军入朝", "中国人民志愿军跨过鸭绿江，第一次战役在云山、温井一带打响。此后三年，数十万年轻人把命留在了那片山地。", "安东，朝鲜", "彭德怀", "战事，志愿军"},
	{1951, 5, 23, "西藏和平解放", "《中央人民政府和西藏地方政府关于和平解放西藏办法的协议》在北京签署。人民解放军随后进藏，雪域高原自此纳入新政权版图。", "北京，拉萨", "阿沛·阿旺晋美", "统一，西藏"},
	{1953, 7, 27, "朝鲜停战协定", "《朝鲜停战协定》于板门店签字。三年战事暂告停歇，三八线依旧，半岛分裂的格局就此固定下来。", "板门店", "彭德怀，克拉克", "战事，停战"},
	{1954, 9, 15, "第一届全国人大", "第一届全国人民代表大会在北京开幕，通过《中华人民共和国宪法》。人民代表大会制度自此确立，国家机构有了新的框架。", "北京", "毛泽东，刘少奇", "建政，宪法"},
	{1955, 4, 18, "万隆会议", "二十九个亚非国家代表聚于万隆。周恩来提出「求同存异」，在会议陷入僵局时把局面拉了回来。新中国的外交，自此有了自己的语气。", "万隆", "周恩来，苏加诺", "外交，亚非"},
	{1956, 7, 13, "第一辆解放牌汽车", "长春第一汽车制造厂驶出第一辆解放牌卡车。从这一天起，中国有了自己的汽车工业；厂区广播里放了一整晚的歌。", "长春", "一汽工人", "工业，建设"},
	{1956, 9, 15, "中共八大", "中国共产党第八次全国代表大会在北京召开，提出国内主要矛盾已变为人民对经济文化迅速发展的需要同当前经济文化不能满足需要之间的矛盾。", "北京", "毛泽东，刘少奇，邓小平", "政党，建设"},
	{1957, 10, 15, "武汉长江大桥通车", "武汉长江大桥建成通车，「一桥飞架南北，天堑变通途」。京汉、粤汉两条铁路自此连成一线。", "武汉", "建桥工人", "工程，交通"},
	{1958, 8, 17, "人民公社化", "北戴河会议通过建立人民公社的决议。此后数月，全国农村几乎一夜之间挂起公社的牌子，公共食堂的炊烟处处升起。", "北戴河，全国", "毛泽东", "农村，公社"},
	{1959, 9, 26, "大庆油田发现", "松基三井喷出工业油流，大庆油田被发现。王进喜带着钻井队住进荒原，「铁人」的名号从此传开。", "黑龙江，大庆", "王进喜", "工业，石油"},
	{1960, 5, 25, "中国人首登珠峰", "中国登山队王富洲、贡布、屈银华从北坡登上珠穆朗玛峰，完成人类首次北坡登顶。氧气将尽，三个人在峰顶只停留了十几分钟。", "珠穆朗玛峰", "王富洲，贡布，屈银华", "登山，纪录"},
	{1962, 10, 20, "中印边界自卫反击", "中印边境东西两段同时发生武装冲突，中国军队推进后主动后撤。此战之后，两国边界问题长期悬而未决。", "中印边境", "张国华", "边事，战事"},
	{1964, 10, 16, "第一颗原子弹", "中国第一颗原子弹在新疆罗布泊爆炸成功。蘑菇云升起时，试验场一片欢呼；北京方面，只发了一条简短的消息。", "罗布泊", "钱学森，邓稼先", "国防，两弹"},
	{1966, 5, 16, "五一六通知", "中共中央政治局扩大会议通过《五一六通知》，「文化大革命」全面发动。此后十年，学校停课，工厂停工，无数人的命运被改写。", "北京", "毛泽东", "文革，政治"},
	{1967, 6, 17, "第一颗氢弹", "中国第一颗氢弹在罗布泊空爆成功。从原子弹到氢弹，中国用了两年零八个月。", "罗布泊", "于敏，邓稼先", "国防，两弹"},
	{1970, 4, 24, "东方红一号", "中国第一颗人造地球卫星「东方红一号」发射成功，卫星在轨播放《东方红》乐曲。中国成为第五个独立发射卫星的国家。", "酒泉", "钱学森，孙家栋", "航天，卫星"},
	{1971, 10, 25, "恢复联合国席位", "联合国大会通过第 2758 号决议，恢复中华人民共和国在联合国的一切合法权利。代表团抵达纽约时，会场掌声持续了很久。", "纽约", "乔冠华", "外交，联合国"},
	{1972, 2, 21, "尼克松访华", "美国总统尼克松抵达北京，中美两国在上海发表《联合公报》。握手的一刻，冷战格局裂开一道缝。", "北京，上海", "尼克松，周恩来", "外交，中美"},
	{1976, 1, 8, "周恩来逝世", "周恩来在北京逝世，享年七十八岁。灵车经过长安街时，数十万市民在寒风中自发送行。", "北京", "周恩来", "人物，哀悼"},
	{1976, 7, 28, "唐山地震", "唐山发生 7.8 级地震，二十四万余人遇难。整座城市在凌晨的几分钟里被抹平，救援部队徒步进入废墟。", "唐山", "受灾群众，救援部队", "灾难，地震"},
	{1976, 9, 9, "毛泽东逝世", "毛泽东在北京逝世，享年八十三岁。全国设灵堂，工厂、学校、街道同时停下。", "北京", "毛泽东", "人物，哀悼"},
	{1976, 10, 6, "粉碎四人帮", "中共中央政治局对江青、张春桥、姚文元、王洪文实行隔离审查，「文化大革命」至此结束。各地群众自发上街游行。", "北京", "华国锋，叶剑英", "政治，转折"},
	{1977, 10, 21, "恢复高考", "国务院决定恢复高考。消息公布后，积压了十年的考生翻出课本，在田间、车间和宿舍里重新演算。那年冬天，五百七十万人走进考场。", "全国", "邓小平", "教育，高考"},
	{1978, 12, 18, "十一届三中全会", "中共十一届三中全会在北京召开，作出把工作重点转移到社会主义现代化建设上来的决定。改革开放的大幕，自此拉开。", "北京", "邓小平，陈云", "改革，转折"},
	{1979, 1, 1, "中美建交", "中美两国正式建立外交关系。同日，全国人大常委会发表《告台湾同胞书》，宣布停止对金门等岛屿的炮击。", "北京，华盛顿", "邓小平，卡特", "外交，中美"},
	{1980, 8, 26, "深圳经济特区", "全国人大常委会批准在深圳、珠海、汕头、厦门设置经济特区。深圳河边的小渔村，开始变成一个大工地。", "深圳", "特区建设者", "改革，特区"},
	{1981, 11, 16, "中国女排首夺世界杯", "中国女排在日本大阪夺得第三届世界杯冠军。这是中国三大球第一个世界冠军，校园里敲脸盆、举火把的人挤满了操场。", "大阪", "郎平，袁伟民", "体育，女排"},
	{1984, 10, 1, "国庆三十五周年", "建国三十五周年阅兵在天安门广场举行。北京大学学生打出「小平您好」的横幅，成为那天最被记住的画面。", "北京", "邓小平，受阅部队", "国庆，改革"},
	{1985, 5, 23, "百万大裁军", "中央军委决定裁减军队员额一百万。铁道兵、基建工程兵集体转业，很多部队在整编中撤编。", "北京", "邓小平", "军队，改革"},
	{1987, 4, 13, "中葡联合声明", "中葡两国在北京签署关于澳门问题的联合声明，宣布澳门将于 1999 年 12 月 20 日回归中国。", "北京，里斯本", "邓小平，席尔瓦", "外交，澳门"},
	{1989, 5, 15, "戈尔巴乔夫访华", "苏联领导人戈尔巴乔夫抵达北京，中苏两国关系实现正常化。相隔三十年，两个最大的社会主义国家重新握手。", "北京", "戈尔巴乔夫，邓小平", "外交，中苏"},
	{1990, 9, 22, "北京亚运会", "第十一届亚运会在北京开幕。这是中国第一次承办综合性国际运动会，吉祥物「盼盼」贴满了大街小巷。", "北京", "运动员，志愿者", "体育，亚运"},
	{1990, 11, 26, "上海证券交易所开业", "上海证券交易所正式开业。开业初期只有八只股票，柜台前排队的人从外滩一直绕到街角。", "上海", "交易所员工", "金融，证券"},
	{1991, 12, 26, "苏联解体", "苏联最高苏维埃通过决议，宣布苏联停止存在。冷战结束，中国北方的漫长边境线，一夜之间换了邻居。", "莫斯科", "戈尔巴乔夫，叶利钦", "世界，冷战"},
	{1992, 1, 18, "邓小平南巡", "邓小平先后到武昌、深圳、珠海、上海等地视察，沿途发表谈话，强调「发展才是硬道理」。新一轮改革由此启动。", "武昌，深圳，珠海，上海", "邓小平", "改革，南巡"},
	{1994, 12, 14, "三峡工程开工", "长江三峡水利枢纽工程正式开工。此后十余年，百万移民迁离故土，江水改道，高峡出平湖。", "宜昌", "建设者，移民", "工程，三峡"},
	{1995, 5, 6, "实行双休日", "国务院决定自 5 月 1 日起实行每周五天工作制。中国人的周末，第一次有了完整的两天。", "全国", "上班族", "社会，生活"},
	{1997, 7, 1, "香港回归", "中国政府对香港恢复行使主权。交接仪式在会展中心举行，米字旗降下，五星红旗升起。维多利亚港的烟花，映了整夜。", "香港", "江泽民，查尔斯", "回归，香港"},
	{1998, 8, 7, "九八抗洪", "长江九江段决口，数万军民在堤上昼夜封堵。那年夏天，全国有二十九个省区市遭受洪灾，抗洪成了整个国家的记忆。", "九江，长江流域", "抗洪军民", "灾难，洪水"},
	{1999, 12, 20, "澳门回归", "中国政府对澳门恢复行使主权。零时，澳门文化中心花园馆内国旗升起，《七子之歌》在街头传唱。", "澳门", "何厚铧", "回归，澳门"},
	{2001, 7, 13, "北京申奥成功", "国际奥委会在莫斯科宣布北京获得 2008 年奥运会主办权。消息传回，天安门广场和长安街瞬间被人群填满。", "莫斯科，北京", "何振梁，申奥代表团", "体育，奥运"},
	{2001, 12, 11, "加入世界贸易组织", "中国正式成为世界贸易组织成员。此后二十年，中国制造沿着新的规则进入全球市场，也改变了无数沿海城市的命运。", "日内瓦，北京", "石广生", "经济，入世"},
	{2003, 9, 10, "振兴东北", "国务院常务会议研究实施东北地区等老工业基地振兴战略。国企改制、棚户区改造与资源型城市转型，自此在关外铺开。", "北京，东北", "国务院", "经济，东北"},
	{2003, 10, 15, "神舟五号", "神舟五号载人飞船在酒泉发射，杨利伟成为中国首位进入太空的航天员。飞船绕地球十四圈后返回，内蒙古草原上的回收队等了整整一夜。", "酒泉，内蒙古", "杨利伟", "航天，载人"},
	{2005, 10, 12, "神舟六号", "神舟六号载人飞船发射，费俊龙、聂海胜在轨飞行五天。中国载人航天由「一人一天」进入「多人多天」。", "酒泉", "费俊龙，聂海胜", "航天，载人"},
	{2006, 7, 1, "青藏铁路通车", "青藏铁路全线通车，火车第一次驶上世界屋脊。格尔木到拉萨，一千一百多公里，最高处海拔五千多米。", "格尔木，拉萨", "铁路建设者", "工程，西藏"},
	{2007, 10, 24, "嫦娥一号", "嫦娥一号探月卫星在西昌发射，中国首次月球探测工程启动。卫星传回的第一幅月面图像，出现在次月的新闻里。", "西昌", "航天工程师", "航天，探月"},
	{2008, 5, 12, "汶川地震", "四川汶川发生 8.0 级地震，近七万人遇难。救援队从全国涌向川北，废墟上出现了无数志愿者。", "汶川，北川", "受灾群众，救援队", "灾难，地震"},
	{2008, 8, 8, "北京奥运", "第二十九届夏季奥运会在北京开幕。鸟巢上空的脚印烟花、李宁点燃主火炬，成为那一年的国家记忆。", "北京", "运动员，志愿者", "体育，奥运"},
	{2009, 10, 1, "国庆六十周年", "建国六十周年阅兵在天安门广场举行。受阅方队与群众游行，把六十年压缩成一个上午。", "北京", "受阅部队", "国庆，阅兵"},
	{2010, 5, 1, "上海世博", "第四十一届世界博览会在上海开幕，主题「城市，让生活更美好」。黄浦江两岸的园区，在半年里接待了七千多万人次。", "上海", "参展方，志愿者", "世博，城市"},
	{2011, 2, 22, "利比亚撤侨", "利比亚局势动荡，中国启动大规模撤侨行动，海陆空并用，十余天撤出三万五千余人。这是新中国成立以来最大规模的海外撤侨。", "利比亚，希腊，马耳他", "撤侨工作人员", "外交，撤侨"},
	{2012, 11, 8, "中共十八大", "中国共产党第十八次全国代表大会在北京召开，提出全面建成小康社会。新一届中央领导集体亮相。", "北京", "习近平", "政党，转折"},
	{2013, 9, 7, "一带一路", "习近平在哈萨克斯坦提出共建丝绸之路经济带，次月在印度尼西亚提出二十一世纪海上丝绸之路。此后十年，这个倡议改写了中国与世界的连接方式。", "阿斯塔纳，雅加达", "习近平", "外交，一带一路"},
	{2015, 9, 3, "九三阅兵", "纪念中国人民抗日战争暨世界反法西斯战争胜利七十周年大会与阅兵在天安门广场举行。抗战老兵方队经过时，观礼台上很多人起立。", "北京", "抗战老兵，受阅部队", "阅兵，历史"},
	{2015, 10, 5, "屠呦呦获诺奖", "屠呦呦获得诺贝尔生理学或医学奖，成为首位获得诺贝尔科学奖项的中国本土科学家。青蒿素的故事，从实验室走到了世界面前。", "斯德哥尔摩，北京", "屠呦呦", "科学，诺奖"},
	{2016, 9, 4, "G20 杭州峰会", "二十国集团领导人峰会在杭州举行。西湖的夜色里，中国第一次以主席国身份主持全球经济治理议程。", "杭州", "习近平，与会领导人", "外交，G20"},
	{2017, 5, 5, "C919 首飞", "国产大型客机 C919 在上海浦东机场首飞成功。飞机尾翼上的绿色，在跑道上滑过时很多人举起了手机。", "上海", "C919 团队", "航空，制造"},
	{2017, 10, 18, "中共十九大", "中国共产党第十九次全国代表大会在北京召开，提出中国特色社会主义进入新时代。", "北京", "习近平", "政党，新时代"},
	{2018, 10, 24, "港珠澳大桥", "港珠澳大桥正式通车。这座跨海大桥把香港、珠海、澳门连成一线，车程从三个小时缩短到四十五分钟。", "香港，珠海，澳门", "大桥建设者", "工程，湾区"},
	{2019, 1, 3, "嫦娥四号月背着陆", "嫦娥四号探测器在月球背面冯·卡门撞击坑着陆，人类探测器首次实现月背软着陆。", "月球，北京", "航天工程师", "航天，探月"},
	{2019, 10, 1, "国庆七十周年", "建国七十周年阅兵与群众游行在天安门广场举行。晚上的联欢活动，烟花在长安街上空连放了一个小时。", "北京", "受阅部队，群众", "国庆，阅兵"},
	{2020, 1, 23, "武汉封城", "武汉市疫情防控指挥部发布通告，全市城市公交、地铁、轮渡、长途客运暂停运营，离汉通道关闭。此后七十六天，这座千万人口的城市停了下来。", "武汉", "武汉市民，医护人员", "疫情，封城"},
	{2020, 12, 17, "嫦娥五号返回", "嫦娥五号返回器携带月球样品在内蒙古四子王旗着陆。中国成为第三个从月球采样返回的国家。", "内蒙古，月球", "航天工程师", "航天，探月"},
	{2021, 7, 1, "建党百年", "庆祝中国共产党成立一百周年大会在天安门广场举行。战机编队飞过，广场上的欢呼声压过了引擎。", "北京", "习近平，党员群众", "政党，百年"},
	{2021, 10, 16, "神舟十三", "神舟十三号载人飞船发射，翟志刚、王亚平、叶光富进驻中国空间站，在轨驻留六个月。王亚平成为中国首位出舱的女航天员。", "酒泉，空间站", "翟志刚，王亚平，叶光富", "航天，空间站"},
	{2022, 2, 4, "北京冬奥", "第二十四届冬季奥运会在北京开幕。开幕式上的「二十四节气」倒计时与雪花火炬，让世界看到另一种中国式浪漫。", "北京，张家口", "运动员，志愿者", "体育，冬奥"},
	{2022, 10, 16, "中共二十大", "中国共产党第二十次全国代表大会在北京召开，提出以中国式现代化全面推进中华民族伟大复兴。", "北京", "习近平", "政党，现代化"},
	{2023, 5, 28, "C919 商业首航", "C919 大型客机执飞 MU9191 航班，从上海虹桥飞抵北京首都机场。国产大飞机第一次载客飞上蓝天。", "上海，北京", "机组，乘客", "航空，制造"},
	{2024, 6, 25, "嫦娥六号月背采样", "嫦娥六号返回器携带月背样品在内蒙古着陆，人类首次实现月球背面采样返回。", "内蒙古，月球", "航天工程师", "航天，探月"},
	{2025, 3, 12, "八项规定学习教育", "全国两会后，全党开展深入贯彻中央八项规定精神学习教育。文件很短，要求很具体：开会、发文、调研、接待，都要照着改。", "全国", "党员干部", "政党，作风"},
	{2025, 3, 14, "哪吒之魔童闹海", "《哪吒之魔童闹海》全球票房突破一百五十亿元，登顶全球动画电影票房榜。影院里排队的人，从春节一直排到三月。", "全国，海外", "动画团队，观众", "文化，电影"},
	{2025, 4, 24, "神舟二十号", "神舟二十号载人飞船发射，中国空间站进入常态化运营。此后半年，神舟二十一、二十二号相继飞天，长征系列完成第五百次发射。", "酒泉", "航天员，工程师", "航天，空间站"},
	{2025, 7, 11, "西夏陵申遗", "西夏陵被列入《世界遗产名录》。同年，故宫博物院迎来建院一百周年，赫哲族伊玛堪转入人类非遗代表作名录。", "银川，北京", "文物工作者", "文化，遗产"},
	{2025, 8, 20, "西藏自治区成立六十周年", "西藏自治区成立六十周年庆祝大会在拉萨举行。布达拉宫广场上，彩旗与经幡一起被风吹起。", "拉萨", "各族群众", "民族，西藏"},
	{2025, 8, 31, "上合天津峰会", "上海合作组织成员国元首理事会第二十五次会议在天津举行。会议提出全球治理倡议，并通过多份合作文件。", "天津", "与会领导人", "外交，上合"},
	{2025, 9, 3, "抗战胜利八十周年", "纪念中国人民抗日战争暨世界反法西斯战争胜利八十周年大会与阅兵在天安门广场举行，六十一国领导人出席。", "北京", "抗战老兵，受阅部队", "阅兵，历史"},
	{2025, 9, 25, "新疆自治区成立七十周年", "新疆维吾尔自治区成立七十周年庆祝大会在乌鲁木齐举行。天山南北，歌声与手鼓响成一片。", "乌鲁木齐", "各族群众", "民族，新疆"},
	{2025, 10, 20, "二十届四中全会", "中共二十届四中全会在北京召开，通过关于制定「十五五」规划的建议。未来五年的产业与科技路线，在这份文件里排好了序。", "北京", "中央委员会", "政党，规划"},
	{2025, 10, 24, "台湾光复纪念日", "全国人大常委会决定，将 10 月 25 日设立为台湾光复纪念日。次日，纪念台湾光复八十周年大会在北京举行。", "北京", "全国人大常委会", "历史，台湾"},
	{2025, 11, 9, "第十五届全运会", "第十五届全国运动会在广东、香港、澳门联合举办。这是全运会第一次由粤港澳三地共同承办。", "广州，香港，澳门", "运动员", "体育，全运"},
	{2025, 11, 5, "福建舰入列", "我国第三艘航空母舰福建舰在海南三亚某军港入列，人民海军进入三航母时代。", "三亚", "海军官兵", "国防，海军"},
	{2025, 12, 18, "海南自贸港封关", "海南自由贸易港正式启动全岛封关运作。海关监管、税收政策与贸易规则，在这一天同时切换。", "海南", "海关，建设者", "经济，自贸港"},
	{2026, 2, 6, "米兰冬奥会", "第二十五届冬季奥运会在米兰-科尔蒂纳开幕。中国代表团获五枚金牌，创境外冬奥最佳；徐梦桃、王心迪成为首对同届同项目夺金的夫妻。", "米兰，科尔蒂纳", "中国冬奥代表团", "体育，冬奥"},
	{2026, 3, 5, "十四届全国人大四次会议", "李强作政府工作报告，提出 2026 年国内生产总值增长 4.5% 至 5%，城镇新增就业一千二百万以上，居民消费价格涨幅 2% 左右。", "北京", "李强，人大代表", "两会，经济"},
	{2026, 7, 16, "上半年经济数据", "国家统计局发布数据，2026 年上半年国内生产总值六十九点五七万亿元，同比增长 4.7%。信息传输、软件和信息技术服务业增长 10.7%。", "北京", "国家统计局", "经济，数据"},
	{2026, 7, 20, "美加墨世界杯", "世界杯决赛在纽约落幕，西班牙 1:0 击败阿根廷，队史第二次夺冠。罗德里获金球奖，姆巴佩以十球获金靴奖。", "纽约，多伦多，墨西哥城", "西班牙队，阿根廷队", "体育，世界杯"},
	{2026, 8, 20, "嫦娥七号出征", "嫦娥七号探测器完成出厂，准备奔赴月球南极。中国探月工程进入「驻月时代」的准备阶段。", "北京，文昌", "航天工程师", "航天，探月"},
	{2026, 10, 5, "诺贝尔奖揭晓", "2026 年诺贝尔奖陆续揭晓。华人科学家薛其坤、叶军、刘如谦的名字，在物理、化学奖项的讨论中被反复提起。", "斯德哥尔摩", "薛其坤，叶军，刘如谦", "科学，诺奖"},
	{2026, 10, 23, "台风蔷琵", "第三十号台风「蔷琵」逼近闽浙沿海，动能强、水汽足、雨强大，风雨潮三碰头。水利部专题部署暴雨洪水防御。", "福建，浙江", "防汛人员", "天气，台风"},
	{2026, 10, 25, "巴青地震", "西藏那曲巴青县发生 7.0 级地震，首批六十套安置板房连夜搭建完成。当晚 19 时 25 分，又发生 5.7 级余震。", "那曲，巴青", "救援队，受灾群众", "灾难，地震"},
	{2026, 10, 26, "渝昆高铁铺轨贯通", "渝昆高铁全线铺轨完成，昆明至重庆将缩短至约两个半小时。同日，二十届中央第七轮巡视反馈完成，上证系列综合指数启用新的编制方案。", "昆明，重庆", "铁路建设者", "工程，交通"},
}

var branchSeeds = []seedBranch{
	{
		Name:        "别史·新潮",
		Description: "从恢复高考处另起一脉，专记考场、校园、刊物与夜谈。正史写谁掌权，别史写谁改变了语言。",
		Official:    true,
		Parent:      "正史·共和本纪",
		ForkAt:      "恢复高考",
		Events: []seedEvent{
			{1978, 3, 1, "首批新生入学", "恢复高考后的第一批大学生走进校园。班里年龄最大的三十多岁，最小的十六岁；有人带着孩子来报到。", "北京，全国", "七七级新生", "教育，校园"},
			{1980, 10, 1, "校园诗社", "北大、复旦等校的诗社相继恢复，油印刊物在宿舍间传阅。诗和争论，成了那几年校园里最便宜也最奢侈的东西。", "北京，上海", "校园诗人", "文学，校园"},
			{1983, 2, 12, "第一届春晚", "中央电视台第一次直播春节联欢晚会。李谷一一人唱了七首歌，热线电话被打爆；此后的年夜饭，多了一道背景音。", "北京", "李谷一，观众", "文化，电视"},
			{1985, 5, 1, "文化热", "「文化：中国与世界」丛书出版，校园里人人都在谈萨特、海德格尔与《走向未来》。书店门口排队的人，比电影院还多。", "北京，上海", "学者，学生", "思想，丛书"},
			{1987, 11, 12, "肯德基前门开业", "肯德基第一家中国门店在北京前门开业。开业当天排队的人绕了三个街口，一份套餐的价钱抵得上普通工人几天工资。", "北京", "北京市民", "生活，开放"},
		},
	},
	{
		Name:        "别史·实业志",
		Description: "自南巡分出的经济一线：分税、国企、入世与制造。正史记政策，这里记账本。",
		Official:    true,
		Parent:      "正史·共和本纪",
		ForkAt:      "邓小平南巡",
		Events: []seedEvent{
			{1994, 1, 1, "分税制改革", "中央与地方财政分税制改革全面推行。此后二十年，土地财政、开发区与地方债，都从这套账本里长出来。", "全国", "财税干部", "经济，财政"},
			{1998, 3, 1, "国企三年脱困", "国企改革进入攻坚阶段，纺织行业压锭、煤炭行业关井，数千万职工下岗再就业。很多厂门口，挂起了「再就业服务中心」的牌子。", "全国，东北", "国企职工", "经济，国企"},
			{2001, 12, 11, "义乌外贸", "中国入世后，义乌小商品市场的外商越来越多。一个县级市，把货卖到了一百多个国家；报关行的小楼里，堆满了英文单证。", "义乌", "商户，外商", "经济，外贸"},
			{2010, 1, 21, "制造业第一", "中国制造业增加值首次超过美国，成为全球第一制造大国。珠三角的流水线，昼夜不停地为世界供货。", "全国，珠三角", "产业工人", "经济，制造"},
			{2020, 9, 1, "双循环", "中央提出构建以国内大循环为主体、国内国际双循环相互促进的新发展格局。外贸企业开始同时盯着两个市场。", "全国", "企业家，外贸商", "经济，格局"},
		},
	},
	{
		Name:        "野史·沪上旧梦",
		Description: "正史写到浦东便翻页，这里偏要留在上海。股票、地铁、高楼与弄堂口的风，都是史。",
		Owner:       "shuoshuren",
		Parent:      "正史·共和本纪",
		ForkAt:      "上海证券交易所开业",
		Events: []seedEvent{
			{1992, 1, 19, "股票认购证", "上海发行股票认购证，三十元一本。最初无人问津，后来黑市炒到上千元；一张纸，改写了无数上海家庭的账本。", "上海", "股民，证券公司", "金融，股票"},
			{1995, 4, 10, "地铁一号线", "上海地铁一号线全线通车。从锦江乐园到上海火车站，地面上的梧桐与地下的灯光，第一次连成了一条线。", "上海", "地铁建设者", "市政，交通"},
			{1999, 8, 28, "金茂大厦", "金茂大厦落成，四百二十米的高度刷新上海天际线。电梯以每秒九米的速度上升，游客在观光层排队看黄浦江。", "上海", "建设者", "建筑，浦东"},
			{2001, 10, 21, "APEC 上海", "亚太经合组织第九次领导人非正式会议在上海举行。与会领导人穿着唐装合影，外滩的灯光亮了一整夜。", "上海", "与会领导人，志愿者", "外交，APEC"},
			{2010, 5, 1, "世博动迁", "上海世博会开幕，浦江两岸的工厂与码头变成展馆。动迁居民搬进郊区新居，老弄堂在推土机下消失。", "上海", "动迁居民，志愿者", "城市，世博"},
		},
	},
	{
		Name:        "野史·关外纪事",
		Description: "从振兴东北分叉。国企、棚改、高铁与雪，关外几十年的热闹与冷清，都记在这里。",
		Owner:       "baishanke",
		Parent:      "正史·共和本纪",
		ForkAt:      "振兴东北",
		Events: []seedEvent{
			{2004, 8, 1, "棚户区改造", "辽宁抚顺、阜新等地的棚户区开始成片改造。矿工们从低矮的平房搬进楼房，老邻居被分到不同的小区。", "抚顺，阜新", "矿工家庭", "民生，棚改"},
			{2008, 8, 1, "哈大高铁", "哈大高速铁路开工建设。此后，沈阳、长春、哈尔滨之间的时间被压缩到两个小时以内。", "哈尔滨，大连", "铁路建设者", "工程，高铁"},
			{2018, 1, 1, "雪乡", "黑龙江雪乡的游客数量创下新高。雪蘑菇、火炕与冻梨，在社交平台上被反复拍摄；也有人抱怨价格。", "牡丹江", "游客，商户", "旅游，关外"},
			{2023, 12, 18, "尔滨", "哈尔滨冰雪大世界开园，南方游客涌入。冻梨摆盘、人造月亮与「尔滨」的称呼，让这座老工业城市在冬天火了一次。", "哈尔滨", "游客，市民", "旅游，城市"},
		},
	},
	{
		Name:        "野史·三线纪事",
		Description: "从第一颗原子弹分叉。山沟里的工厂、地图上找不到的代号，和一代人的青春。",
		Owner:       "tingyuzhe",
		Parent:      "正史·共和本纪",
		ForkAt:      "第一颗原子弹",
		Events: []seedEvent{
			{1965, 3, 1, "攀枝花", "攀枝花钢铁基地开工建设。来自全国的建设者住进干打垒，在金沙江边的荒山上炼出第一炉铁。", "攀枝花", "三线建设者", "工业，三线"},
			{1966, 7, 1, "成昆铁路", "成昆铁路全线贯通。一千多公里，穿过大小凉山，平均每公里就有一名建设者牺牲。", "成都，昆明", "铁道兵", "工程，三线"},
			{1970, 4, 24, "戈壁上的广播", "东方红一号飞过西北上空时，基地的广播里传出《东方红》。山沟里的职工端着饭碗站在操场上，抬头看天。", "酒泉，西北", "基地职工", "航天，三线"},
			{1980, 1, 1, "出山", "三线企业开始调整搬迁，部分工厂从山沟迁往城市。老厂区留下的车间、宿舍和标语，慢慢长满了草。", "西南，西北", "三线职工", "三线，转型"},
		},
	},
	{
		Name:        "野史·金陵旧事",
		Description: "从中美建交分叉。一座城的八九十年代：饭店、秦淮河、倒计时与青奥。",
		Owner:       "jinling",
		Parent:      "正史·共和本纪",
		ForkAt:      "中美建交",
		Events: []seedEvent{
			{1983, 5, 1, "金陵饭店", "金陵饭店建成开业，三十七层的高度在当时的南京城里一枝独秀。旋转餐厅转一圈四十分钟，能看见整座城墙。", "南京", "饭店员工", "建筑，南京"},
			{1988, 6, 1, "秦淮河整治", "秦淮河整治工程启动，沿岸的排污口被逐个封堵。画舫重新下水时，夫子庙的灯会挤得走不动路。", "南京", "市政工人，市民", "市政，南京"},
			{1997, 6, 30, "夫子庙的倒计时", "香港回归前夜，夫子庙广场上支起大屏幕。零点倒数时，人群齐声喊出数字，秦淮河上的画舫同时鸣笛。", "南京", "南京市民", "回归，南京"},
			{2014, 8, 16, "南京青奥会", "第二届夏季青年奥林匹克运动会在南京开幕。青奥村建在河西，开幕式上的「筑梦」让很多南京人第一次觉得，河西也是市中心。", "南京", "运动员，志愿者", "体育，青奥"},
		},
	},
	{
		Name:        "野史·南粤潮声",
		Description: "从深圳经济特区分叉。国贸、罗湖桥、广州亚运，南风一吹，日子就变了。",
		Owner:       "lingnan",
		Parent:      "正史·共和本纪",
		ForkAt:      "深圳经济特区",
		Events: []seedEvent{
			{1983, 10, 1, "深圳国贸大厦", "深圳国贸大厦动工，施工队创下「三天一层楼」的速度。国贸封顶时，深圳河边还是一片工地。", "深圳", "建设者", "建设，深圳"},
			{1992, 1, 19, "春风吹过深圳湾", "邓小平南巡抵达深圳，登上国贸大厦旋转餐厅。窗外是正在长高的城市，他说：「深圳发展这么快，是靠实干干出来的。」", "深圳", "邓小平", "改革，深圳"},
			{1997, 6, 30, "罗湖桥的倒计时", "香港回归前夜，罗湖桥两侧挤满了人。驻港部队车队在雨中驶过，桥头的哨兵换了岗。", "深圳，香港", "驻港部队，市民", "回归，深圳"},
			{2010, 11, 12, "广州亚运会", "第十六届亚运会在广州开幕。开幕式在珠江上的海心沙举行，以水为舞台；此后，珠江夜游的船票一直紧张。", "广州", "运动员，志愿者", "体育，亚运"},
		},
	},
}
