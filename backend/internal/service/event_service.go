package service

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"gonghe/internal/config"
	"gonghe/internal/model"

	"gorm.io/gorm"
)

// EventService 负责条目的读取、续写与删改。
type EventService struct {
	db       *gorm.DB
	cfg      *config.Config
	branches *BranchService
}

func NewEventService(db *gorm.DB, cfg *config.Config, branches *BranchService) *EventService {
	return &EventService{db: db, cfg: cfg, branches: branches}
}

// EventFilter 是全站条目流的查询条件。
type EventFilter struct {
	Scope    string // all / official / folk
	Sort     string // latest / hot
	Q        string
	Page     int
	PageSize int
}

// Feed 返回全站最新条目，跨史枝合并。
func (s *EventService) Feed(viewerID uint64, f EventFilter) ([]TimelineItem, int64, error) {
	page, size := clampPage(f.Page, f.PageSize, s.cfg.DefaultPageSize, s.cfg.MaxPageSize)

	build := func() *gorm.DB {
		q := s.db.Table("events").Joins("JOIN branches AS b ON b.id = events.branch_id")
		switch f.Scope {
		case "official":
			q = q.Where("b.official = ?", true)
		case "folk":
			q = q.Where("b.official = ?", false)
		}
		if q2 := strings.TrimSpace(f.Q); q2 != "" {
			like := "%" + q2 + "%"
			q = q.Where("(events.title LIKE ? OR events.content LIKE ? OR events.location LIKE ?)", like, like, like)
		}
		return q
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := "events.year DESC, events.month DESC, events.day DESC, events.id DESC"
	switch f.Sort {
	case "hot":
		order = "events.like_count DESC, events.id DESC"
	case "latest":
		order = "events.id DESC"
	}
	var rows []model.Event
	if err := build().Select("events.*").Order(order).
		Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items, err := timelineItems(s.db, rows, viewerID, 0)
	return items, total, err
}

// Detail 读取单条实录。
func (s *EventService) Detail(viewerID, id uint64) (*EventDetail, error) {
	var e model.Event
	if err := s.db.First(&e, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, notFound("条目不存在")
		}
		return nil, err
	}
	b, err := getBranch(s.db, e.BranchID)
	if err != nil {
		return nil, err
	}
	items, err := timelineItems(s.db, []model.Event{e}, viewerID, b.ID)
	if err != nil {
		return nil, err
	}
	item := items[0]
	item.Content = e.Content

	cards, err := s.branches.cards([]model.Branch{*b}, viewerID)
	if err != nil {
		return nil, err
	}
	canEdit := viewerID != 0 && !b.Official && (b.OwnerID == viewerID || e.UserID == viewerID)
	return &EventDetail{TimelineItem: item, Branch: cards[0], CanEdit: canEdit}, nil
}

// EventRequest 是新增 / 修改条目的入参。
type EventRequest struct {
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Year     int      `json:"year"`
	Month    int      `json:"month"`
	Day      int      `json:"day"`
	Location string   `json:"location"`
	Figures  []string `json:"figures"`
	Tags     []string `json:"tags"`
}

func normalizeEventRequest(req *EventRequest) error {
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	req.Location = strings.TrimSpace(req.Location)

	if n := utf8.RuneCountInString(req.Title); n < 2 || n > 60 {
		return badRequest("条目标题需 2–60 字")
	}
	if n := utf8.RuneCountInString(req.Content); n < 1 || n > 20000 {
		return badRequest("条目正文需 1–20000 字")
	}
	if req.Year < 1850 || req.Year > 2100 {
		return badRequest("年份需在 1850–2100 之间")
	}
	if req.Month < 0 || req.Month > 12 {
		return badRequest("月份不合法")
	}
	if req.Day < 0 || req.Day > 31 {
		return badRequest("日期不合法")
	}
	if req.Day > 0 && req.Month == 0 {
		return badRequest("填写日之前请先选择月份")
	}
	if utf8.RuneCountInString(req.Location) > 40 {
		return badRequest("地点不能超过 40 字")
	}
	if len(req.Figures) > 8 {
		return badRequest("人物最多 8 位")
	}
	if len(req.Tags) > 6 {
		return badRequest("标签最多 6 个")
	}
	for _, f := range req.Figures {
		if utf8.RuneCountInString(strings.TrimSpace(f)) > 24 {
			return badRequest("单个人物名不能超过 24 字")
		}
	}
	for _, t := range req.Tags {
		if utf8.RuneCountInString(strings.TrimSpace(t)) > 16 {
			return badRequest("单个标签不能超过 16 字")
		}
	}
	return nil
}

// Create 在史枝末尾续写一条实录。只有枝主能写。
func (s *EventService) Create(userID, branchID uint64, req EventRequest) (*EventDetail, error) {
	if userID == 0 {
		return nil, forbidden("请先登录")
	}
	b, err := getBranch(s.db, branchID)
	if err != nil {
		return nil, err
	}
	if b.Official || b.OwnerID != userID {
		return nil, forbidden("只有史枝主人可以续写；官修正史请先分叉")
	}
	if err := normalizeEventRequest(&req); err != nil {
		return nil, err
	}

	now := time.Now()
	e := model.Event{
		BranchID:  branchID,
		UserID:    userID,
		Title:     req.Title,
		Content:   req.Content,
		Year:      req.Year,
		Month:     req.Month,
		Day:       req.Day,
		Location:  req.Location,
		Figures:   joinList(req.Figures),
		Tags:      joinList(req.Tags),
		CreatedAt: now,
		UpdatedAt: now,
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&e).Error; err != nil {
			return err
		}
		return refreshBranchStats(tx, branchID)
	})
	if err != nil {
		return nil, err
	}
	return s.Detail(userID, e.ID)
}

// Update 修改条目。允许枝主与原作者（正常情况下是同一个人）。
func (s *EventService) Update(userID, id uint64, req EventRequest) (*EventDetail, error) {
	var e model.Event
	if err := s.db.First(&e, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, notFound("条目不存在")
		}
		return nil, err
	}
	b, err := getBranch(s.db, e.BranchID)
	if err != nil {
		return nil, err
	}
	if b.Official || userID == 0 || (b.OwnerID != userID && e.UserID != userID) {
		return nil, forbidden("无权修改此条目")
	}
	if err := normalizeEventRequest(&req); err != nil {
		return nil, err
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Event{}).Where("id = ?", id).Updates(map[string]any{
			"title":    req.Title,
			"content":  req.Content,
			"year":     req.Year,
			"month":    req.Month,
			"day":      req.Day,
			"location": req.Location,
			"figures":  joinList(req.Figures),
			"tags":     joinList(req.Tags),
		}).Error; err != nil {
			return err
		}
		return refreshBranchStats(tx, e.BranchID)
	})
	if err != nil {
		return nil, err
	}
	return s.Detail(userID, id)
}

// Delete 删除条目。已有人自此处分叉时不允许删。
func (s *EventService) Delete(userID, id uint64) error {
	var e model.Event
	if err := s.db.First(&e, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return notFound("条目不存在")
		}
		return err
	}
	b, err := getBranch(s.db, e.BranchID)
	if err != nil {
		return err
	}
	if b.Official || userID == 0 || (b.OwnerID != userID && e.UserID != userID) {
		return forbidden("无权删除此条目")
	}
	var forks int64
	if err := s.db.Model(&model.Branch{}).Where("fork_event_id = ?", id).Count(&forks).Error; err != nil {
		return err
	}
	if forks > 0 {
		return conflict("已有 %d 条史枝自此处分叉，不能删除", forks)
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := deleteTargets(tx, model.TargetEvent, []uint64{id}); err != nil {
			return err
		}
		if err := tx.Delete(&model.Event{}, id).Error; err != nil {
			return err
		}
		return refreshBranchStats(tx, e.BranchID)
	})
}

// Mine 返回当前用户写过的条目。
func (s *EventService) Mine(userID uint64, page, pageSize int) ([]TimelineItem, int64, error) {
	page, size := clampPage(page, pageSize, s.cfg.DefaultPageSize, s.cfg.MaxPageSize)
	var total int64
	if err := s.db.Model(&model.Event{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Event
	if err := s.db.Where("user_id = ?", userID).Order("id DESC").
		Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items, err := timelineItems(s.db, rows, userID, 0)
	return items, total, err
}

// refreshBranchStats 重算一条史枝的自有条目数与本枝年份范围。
// 条目可能被补录到很早的年份，所以不能用「max(旧值, 新值)」。
func refreshBranchStats(tx *gorm.DB, branchID uint64) error {
	var row struct {
		Cnt     int
		MinYear int
		MaxYear int
	}
	if err := tx.Raw(
		`SELECT COUNT(*) AS cnt,
		        COALESCE(MIN(year), 0) AS min_year,
		        COALESCE(MAX(year), 0) AS max_year
		   FROM events WHERE branch_id = ?`, branchID).Scan(&row).Error; err != nil {
		return err
	}
	return tx.Model(&model.Branch{}).Where("id = ?", branchID).Updates(map[string]any{
		"event_count": row.Cnt,
		"first_year":  row.MinYear,
		"last_year":   row.MaxYear,
	}).Error
}

// timelineItems 把条目批量转成前端结构（作者、所属枝、我是否赞过）。
func timelineItems(db *gorm.DB, events []model.Event, viewerID, currentBranchID uint64) ([]TimelineItem, error) {
	out := make([]TimelineItem, 0, len(events))
	if len(events) == 0 {
		return out, nil
	}
	branchIDs := make([]uint64, 0, len(events))
	userIDs := make([]uint64, 0, len(events))
	eventIDs := make([]uint64, 0, len(events))
	for _, e := range events {
		branchIDs = append(branchIDs, e.BranchID)
		userIDs = append(userIDs, e.UserID)
		eventIDs = append(eventIDs, e.ID)
	}
	branches, err := loadBranches(db, branchIDs)
	if err != nil {
		return nil, err
	}
	users, err := loadUsers(db, userIDs)
	if err != nil {
		return nil, err
	}
	liked, err := likedIDs(db, viewerID, model.TargetEvent, eventIDs)
	if err != nil {
		return nil, err
	}

	for _, e := range events {
		item := TimelineItem{
			ID:           e.ID,
			BranchID:     e.BranchID,
			Title:        e.Title,
			Excerpt:      excerpt(e.Content, 110),
			Year:         e.Year,
			Month:        e.Month,
			Day:          e.Day,
			Location:     e.Location,
			Figures:      splitList(e.Figures),
			Tags:         splitList(e.Tags),
			LikeCount:    e.LikeCount,
			CommentCount: e.CommentCount,
			Liked:        liked[e.ID],
			Inherited:    currentBranchID != 0 && e.BranchID != currentBranchID,
			CreatedAt:    e.CreatedAt,
		}
		if b, ok := branches[e.BranchID]; ok {
			item.Branch = b.Name
			item.Official = b.Official
		}
		if u, ok := users[e.UserID]; ok {
			item.Author = briefOf(u)
		} else {
			item.Author = &UserBrief{ID: e.UserID, Nickname: "佚名"}
		}
		out = append(out, item)
	}
	return out, nil
}
