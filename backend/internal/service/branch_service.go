package service

import (
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"gonghe/internal/config"
	"gonghe/internal/model"

	"gorm.io/gorm"
)

// BranchService 负责史枝的读、分叉、改名与删除。
type BranchService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewBranchService(db *gorm.DB, cfg *config.Config) *BranchService {
	return &BranchService{db: db, cfg: cfg}
}

// BranchFilter 是史枝列表的查询条件。
type BranchFilter struct {
	Scope    string // all / official / folk
	Sort     string // hot / new / forks / events
	Q        string
	OwnerID  uint64
	Page     int
	PageSize int
}

// List 返回史枝列表。
func (s *BranchService) List(viewerID uint64, f BranchFilter) ([]BranchCard, int64, error) {
	page, size := clampPage(f.Page, f.PageSize, s.cfg.DefaultPageSize, s.cfg.MaxPageSize)

	build := func() *gorm.DB {
		q := s.db.Model(&model.Branch{})
		switch f.Scope {
		case "official":
			q = q.Where("official = ?", true)
		case "folk":
			q = q.Where("official = ?", false)
		}
		if f.OwnerID > 0 {
			q = q.Where("owner_id = ?", f.OwnerID)
		}
		if q2 := strings.TrimSpace(f.Q); q2 != "" {
			like := "%" + q2 + "%"
			q = q.Where("name LIKE ? OR description LIKE ?", like, like)
		}
		return q
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := "like_count DESC, fork_count DESC, id DESC"
	switch f.Sort {
	case "new":
		order = "id DESC"
	case "forks":
		order = "fork_count DESC, like_count DESC, id DESC"
	case "events":
		order = "(event_count + inherited_count) DESC, like_count DESC, id DESC"
	}

	var rows []model.Branch
	if err := build().Order(order).Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	cards, err := s.cards(rows, viewerID)
	return cards, total, err
}

// Detail 读取单条史枝，附谱系、子枝与同源枝。
func (s *BranchService) Detail(viewerID, id uint64) (*BranchDetail, error) {
	b, err := getBranch(s.db, id)
	if err != nil {
		return nil, err
	}
	cards, err := s.cards([]model.Branch{*b}, viewerID)
	if err != nil {
		return nil, err
	}
	detail := &BranchDetail{
		BranchCard: cards[0],
		IsOwner:    viewerID != 0 && b.OwnerID == viewerID,
	}

	if b.ForkEventID != 0 {
		var e model.Event
		if err := s.db.Select("id", "title").First(&e, b.ForkEventID).Error; err == nil {
			detail.ForkEventTitle = e.Title
		}
	}

	// 向上走谱系。正常数据不会成环，深度上限只是兜底。
	cur := b
	var chain []LineageItem
	for depth := 0; depth < 16 && cur.ParentID != 0; depth++ {
		p, err := getBranch(s.db, cur.ParentID)
		if err != nil {
			break
		}
		chain = append(chain, LineageItem{
			ID:         p.ID,
			Name:       p.Name,
			Official:   p.Official,
			ForkYear:   cur.ForkYear,
			EventCount: p.InheritedCount + p.EventCount,
		})
		cur = p
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	detail.Ancestors = chain
	if detail.Ancestors == nil {
		detail.Ancestors = []LineageItem{}
	}

	var children []model.Branch
	if err := s.db.Where("parent_id = ?", b.ID).Order("id ASC").Find(&children).Error; err != nil {
		return nil, err
	}
	detail.Children, err = s.cards(children, viewerID)
	if err != nil {
		return nil, err
	}
	if detail.Children == nil {
		detail.Children = []BranchCard{}
	}

	if b.ParentID != 0 {
		var siblings []model.Branch
		if err := s.db.Where("parent_id = ? AND id <> ?", b.ParentID, b.ID).
			Order("like_count DESC, id ASC").Limit(4).Find(&siblings).Error; err != nil {
			return nil, err
		}
		detail.Siblings, err = s.cards(siblings, viewerID)
		if err != nil {
			return nil, err
		}
		if detail.Siblings == nil {
			detail.Siblings = []BranchCard{}
		}
	} else {
		detail.Siblings = []BranchCard{}
	}
	return detail, nil
}

// Timeline 返回合并了继承条目的完整时间轴。
func (s *BranchService) Timeline(viewerID, branchID uint64, page, pageSize int, ownOnly bool) ([]TimelineItem, int64, error) {
	if _, err := getBranch(s.db, branchID); err != nil {
		return nil, 0, err
	}
	events, err := s.history(branchID, 0)
	if err != nil {
		return nil, 0, err
	}
	if ownOnly {
		own := make([]model.Event, 0, len(events))
		for _, e := range events {
			if e.BranchID == branchID {
				own = append(own, e)
			}
		}
		events = own
	}
	sort.SliceStable(events, func(i, j int) bool { return eventLess(events[i], events[j]) })

	page, size := clampPage(page, pageSize, s.cfg.DefaultPageSize, s.cfg.MaxPageSize)
	total := int64(len(events))
	startIdx := (page - 1) * size
	if startIdx >= len(events) {
		return []TimelineItem{}, total, nil
	}
	endIdx := min(startIdx+size, len(events))
	items, err := timelineItems(s.db, events[startIdx:endIdx], viewerID, branchID)
	return items, total, err
}

// history 按「提交顺序」取一条史枝的完整历史：父枝前缀 + 自有条目。
//
// 前缀用 ForkEventID 截断，而不是日期。这是 git 的语义：分叉点之后
// 父枝再补录的条目，子枝不继承；即便补录的日期更早。
func (s *BranchService) history(branchID uint64, depth int) ([]model.Event, error) {
	if depth > 24 {
		return nil, conflict("史枝谱系过深，无法展开")
	}
	b, err := getBranch(s.db, branchID)
	if err != nil {
		return nil, err
	}
	var own []model.Event
	if err := s.db.Where("branch_id = ?", branchID).Order("id ASC").Find(&own).Error; err != nil {
		return nil, err
	}
	if b.ParentID == 0 {
		return own, nil
	}

	prefix, err := s.history(b.ParentID, depth+1)
	if err != nil {
		return nil, err
	}
	if b.ForkEventID != 0 {
		idx := -1
		for i := range prefix {
			if prefix[i].ID == b.ForkEventID {
				idx = i
				break
			}
		}
		switch {
		case idx >= 0:
			prefix = prefix[:idx+1]
		case b.InheritedCount < len(prefix):
			prefix = prefix[:b.InheritedCount]
		}
	} else if b.InheritedCount < len(prefix) {
		prefix = prefix[:b.InheritedCount]
	}

	out := make([]model.Event, 0, len(prefix)+len(own))
	out = append(out, prefix...)
	out = append(out, own...)
	return out, nil
}

// ForkRequest 是「开一枝」的入参。
type ForkRequest struct {
	ParentID    uint64 `json:"parentId"`
	ForkEventID uint64 `json:"forkEventId"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Fork 从父枝的某个条目处分叉出一条新史枝。
func (s *BranchService) Fork(userID uint64, req ForkRequest) (*BranchDetail, error) {
	if userID == 0 {
		return nil, forbidden("请先登录")
	}
	name := strings.TrimSpace(req.Name)
	if n := utf8.RuneCountInString(name); n < 2 || n > 40 {
		return nil, badRequest("史枝名需 2–40 字")
	}
	desc := strings.TrimSpace(req.Description)
	if utf8.RuneCountInString(desc) > 300 {
		return nil, badRequest("史枝小引不能超过 300 字")
	}

	parent, err := getBranch(s.db, req.ParentID)
	if err != nil {
		return nil, err
	}
	hist, err := s.history(parent.ID, 0)
	if err != nil {
		return nil, err
	}

	idx := len(hist) - 1
	if req.ForkEventID != 0 {
		idx = -1
		for i := range hist {
			if hist[i].ID == req.ForkEventID {
				idx = i
				break
			}
		}
		if idx < 0 {
			return nil, badRequest("分叉点不在该史枝的时间轴上")
		}
	}
	inherited := idx + 1
	forkYear := 0
	if idx >= 0 {
		forkYear = hist[idx].Year
	}

	var created model.Branch
	err = s.db.Transaction(func(tx *gorm.DB) error {
		created = model.Branch{
			Name:           name,
			Description:    desc,
			OwnerID:        userID,
			Official:       false,
			ParentID:       parent.ID,
			ForkEventID:    req.ForkEventID,
			ForkYear:       forkYear,
			InheritedCount: inherited,
		}
		if err := tx.Create(&created).Error; err != nil {
			return err
		}
		return tx.Model(&model.Branch{}).Where("id = ?", parent.ID).
			UpdateColumn("fork_count", gorm.Expr("fork_count + 1")).Error
	})
	if err != nil {
		return nil, err
	}
	return s.Detail(userID, created.ID)
}

// Update 改名 / 改小引，仅枝主。
func (s *BranchService) Update(userID, id uint64, name, description string) error {
	b, err := getBranch(s.db, id)
	if err != nil {
		return err
	}
	if b.OwnerID == 0 || b.OwnerID != userID {
		return forbidden("只有史枝主人可以修改")
	}
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n < 2 || n > 40 {
		return badRequest("史枝名需 2–40 字")
	}
	description = strings.TrimSpace(description)
	if utf8.RuneCountInString(description) > 300 {
		return badRequest("史枝小引不能超过 300 字")
	}
	return s.db.Model(&model.Branch{}).Where("id = ?", id).Updates(map[string]any{
		"name":        name,
		"description": description,
	}).Error
}

// Delete 删除一条史枝。有人从这里分叉之后就不许删了 —— 不能抽走别人的地基。
func (s *BranchService) Delete(userID, id uint64) error {
	b, err := getBranch(s.db, id)
	if err != nil {
		return err
	}
	if b.OwnerID == 0 || b.OwnerID != userID {
		return forbidden("只有史枝主人可以删除")
	}
	var children int64
	if err := s.db.Model(&model.Branch{}).Where("parent_id = ?", id).Count(&children).Error; err != nil {
		return err
	}
	if children > 0 {
		return conflict("已有 %d 条史枝自此分叉，不能删除", children)
	}

	var eventIDs []uint64
	if err := s.db.Model(&model.Event{}).Where("branch_id = ?", id).Pluck("id", &eventIDs).Error; err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := deleteTargets(tx, model.TargetBranch, []uint64{id}); err != nil {
			return err
		}
		if len(eventIDs) > 0 {
			if err := deleteTargets(tx, model.TargetEvent, eventIDs); err != nil {
				return err
			}
			if err := tx.Where("branch_id = ?", id).Delete(&model.Event{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Delete(&model.Branch{}, id).Error; err != nil {
			return err
		}
		if b.ParentID != 0 {
			return tx.Model(&model.Branch{}).Where("id = ?", b.ParentID).
				UpdateColumn("fork_count", gorm.Expr("CASE WHEN fork_count > 0 THEN fork_count - 1 ELSE 0 END")).Error
		}
		return nil
	})
}

// Graph 返回谱系图的全部节点。
func (s *BranchService) Graph() ([]GraphNode, error) {
	var rows []model.Branch
	if err := s.db.Order("official DESC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	ownerIDs := make([]uint64, 0, len(rows))
	for _, b := range rows {
		ownerIDs = append(ownerIDs, b.OwnerID)
	}
	users, err := loadUsers(s.db, ownerIDs)
	if err != nil {
		return nil, err
	}
	out := make([]GraphNode, 0, len(rows))
	for _, b := range rows {
		start := b.FirstYear
		if start == 0 {
			start = b.ForkYear
		}
		end := b.LastYear
		if end == 0 {
			end = b.ForkYear
		}
		if end < start {
			end = start
		}
		owner := "史馆"
		if b.OwnerID != 0 {
			if u, ok := users[b.OwnerID]; ok {
				owner = u.Nickname
			}
		}
		out = append(out, GraphNode{
			ID:          b.ID,
			Name:        b.Name,
			Official:    b.Official,
			ParentID:    b.ParentID,
			ForkEventID: b.ForkEventID,
			ForkYear:    b.ForkYear,
			StartYear:   start,
			EndYear:     end,
			EventCount:  b.InheritedCount + b.EventCount,
			ForkCount:   b.ForkCount,
			LikeCount:   b.LikeCount,
			Owner:       owner,
		})
	}
	return out, nil
}

// Stats 是首页的统计数字。
func (s *BranchService) Stats() (*Stats, error) {
	st := &Stats{}
	if err := s.db.Model(&model.Event{}).Count(&st.TotalEvents).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.Branch{}).Count(&st.TotalBranches).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.Branch{}).Where("official = ?", true).Count(&st.OfficialBranches).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.Branch{}).Where("official = ?", false).Count(&st.FolkBranches).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.Event{}).
		Joins("JOIN branches AS b ON b.id = events.branch_id").
		Where("b.official = ?", true).Count(&st.OfficialEvents).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.Event{}).
		Joins("JOIN branches AS b ON b.id = events.branch_id").
		Where("b.official = ?", false).Count(&st.FolkEvents).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.Branch{}).
		Where("official = ? AND owner_id > 0", false).
		Distinct("owner_id").Count(&st.Chroniclers).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.Like{}).Count(&st.TotalLikes).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if err := s.db.Model(&model.Event{}).Where("created_at >= ?", startOfDay).Count(&st.TodayEvents).Error; err != nil {
		return nil, err
	}
	return st, nil
}

// cards 把数据库行批量转换成前端卡片，避免逐条查作者 / 父枝 / 点赞状态。
func (s *BranchService) cards(rows []model.Branch, viewerID uint64) ([]BranchCard, error) {
	out := make([]BranchCard, 0, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	ownerIDs := make([]uint64, 0, len(rows))
	ids := make([]uint64, 0, len(rows))
	for _, b := range rows {
		ownerIDs = append(ownerIDs, b.OwnerID)
		ids = append(ids, b.ID)
	}
	users, err := loadUsers(s.db, ownerIDs)
	if err != nil {
		return nil, err
	}
	parentIDs := make([]uint64, 0, len(rows))
	for _, b := range rows {
		parentIDs = append(parentIDs, b.ParentID)
	}
	parents, err := loadBranches(s.db, parentIDs)
	if err != nil {
		return nil, err
	}
	liked, err := likedIDs(s.db, viewerID, model.TargetBranch, ids)
	if err != nil {
		return nil, err
	}

	for _, b := range rows {
		card := branchCardOf(b)
		if b.OwnerID == 0 {
			card.Owner = systemUser()
		} else if u, ok := users[b.OwnerID]; ok {
			card.Owner = briefOf(u)
		} else {
			card.Owner = &UserBrief{ID: b.OwnerID, Nickname: "佚名"}
		}
		if p, ok := parents[b.ParentID]; ok {
			card.ParentName = p.Name
		}
		card.Liked = liked[b.ID]
		card.CanWrite = viewerID != 0 && b.OwnerID == viewerID
		out = append(out, card)
	}
	return out, nil
}

func branchCardOf(b model.Branch) BranchCard {
	return BranchCard{
		ID:             b.ID,
		Name:           b.Name,
		Description:    b.Description,
		Official:       b.Official,
		ParentID:       b.ParentID,
		ForkEventID:    b.ForkEventID,
		ForkYear:       b.ForkYear,
		InheritedCount: b.InheritedCount,
		OwnCount:       b.EventCount,
		EventCount:     b.InheritedCount + b.EventCount,
		ForkCount:      b.ForkCount,
		LikeCount:      b.LikeCount,
		CommentCount:   b.CommentCount,
		FirstYear:      b.FirstYear,
		LastYear:       b.LastYear,
		CreatedAt:      b.CreatedAt,
	}
}

// eventLess 是时间轴的排序：先年份，再月、日，最后按提交顺序。
func eventLess(a, b model.Event) bool {
	if a.Year != b.Year {
		return a.Year < b.Year
	}
	if a.Month != b.Month {
		return a.Month < b.Month
	}
	if a.Day != b.Day {
		return a.Day < b.Day
	}
	return a.ID < b.ID
}
