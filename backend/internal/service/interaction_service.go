package service

import (
	"errors"
	"strings"
	"unicode/utf8"

	"gonghe/internal/config"
	"gonghe/internal/model"

	"gorm.io/gorm"
)

// InteractionService 负责点赞与评论 —— 两种最轻的「读者反应」。
type InteractionService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewInteractionService(db *gorm.DB, cfg *config.Config) *InteractionService {
	return &InteractionService{db: db, cfg: cfg}
}

// targetTable 把对象类型映射到表名。只用内部常量，不接受外部表名。
func targetTable(targetType string) (string, bool) {
	switch targetType {
	case model.TargetBranch:
		return "branches", true
	case model.TargetEvent:
		return "events", true
	case model.TargetComment:
		return "comments", true
	default:
		return "", false
	}
}

func isCommentable(targetType string) bool {
	return targetType == model.TargetBranch || targetType == model.TargetEvent
}

func targetCount(db *gorm.DB, targetType string, id uint64) (int64, error) {
	table, ok := targetTable(targetType)
	if !ok {
		return 0, badRequest("未知的对象类型")
	}
	var n int64
	if err := db.Table(table).Where("id = ?", id).Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

func adjustTargetCount(tx *gorm.DB, targetType string, id uint64, column string, delta int) error {
	table, ok := targetTable(targetType)
	if !ok {
		return badRequest("未知的对象类型")
	}
	if delta < 0 {
		return tx.Table(table).Where("id = ?", id).
			UpdateColumn(column, gorm.Expr("CASE WHEN "+column+" > 0 THEN "+column+" - 1 ELSE 0 END")).Error
	}
	return tx.Table(table).Where("id = ?", id).
		UpdateColumn(column, gorm.Expr(column+" + 1")).Error
}

func readTargetCount(db *gorm.DB, targetType string, id uint64, column string) (int, error) {
	table, ok := targetTable(targetType)
	if !ok {
		return 0, badRequest("未知的对象类型")
	}
	var n int
	if err := db.Table(table).Select(column).Where("id = ?", id).Scan(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// ToggleLike 切换点赞状态，返回切换后的状态与总数。
func (s *InteractionService) ToggleLike(userID uint64, targetType string, targetID uint64) (bool, int, error) {
	if userID == 0 {
		return false, 0, forbidden("请先登录")
	}
	if _, ok := targetTable(targetType); !ok {
		return false, 0, badRequest("未知的对象类型")
	}
	n, err := targetCount(s.db, targetType, targetID)
	if err != nil {
		return false, 0, err
	}
	if n == 0 {
		return false, 0, notFound("点赞对象不存在")
	}

	liked := false
	count := 0
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var like model.Like
		findErr := tx.Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
			First(&like).Error
		switch {
		case findErr == nil:
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			if err := adjustTargetCount(tx, targetType, targetID, "like_count", -1); err != nil {
				return err
			}
			liked = false
		case errors.Is(findErr, gorm.ErrRecordNotFound):
			newLike := model.Like{UserID: userID, TargetType: targetType, TargetID: targetID}
			if err := tx.Create(&newLike).Error; err != nil {
				if !errors.Is(err, gorm.ErrDuplicatedKey) {
					return err
				}
				// 并发重复点赞：视为已赞，不再加计数
				liked = true
			} else {
				liked = true
				if err := adjustTargetCount(tx, targetType, targetID, "like_count", 1); err != nil {
					return err
				}
			}
		default:
			return findErr
		}
		var readErr error
		count, readErr = readTargetCount(tx, targetType, targetID, "like_count")
		return readErr
	})
	if err != nil {
		return false, 0, err
	}
	return liked, count, nil
}

// Comments 读取一个对象下的评论，按时间倒序。
func (s *InteractionService) Comments(viewerID uint64, targetType string, targetID uint64, page, pageSize int) ([]CommentView, int64, error) {
	if !isCommentable(targetType) {
		return nil, 0, badRequest("该对象不支持评论")
	}
	page, size := clampPage(page, pageSize, s.cfg.DefaultPageSize, s.cfg.MaxPageSize)

	q := s.db.Model(&model.Comment{}).Where("target_type = ? AND target_id = ?", targetType, targetID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Comment
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	userIDs := make([]uint64, 0, len(rows))
	ids := make([]uint64, 0, len(rows))
	for _, c := range rows {
		userIDs = append(userIDs, c.UserID)
		ids = append(ids, c.ID)
	}
	users, err := loadUsers(s.db, userIDs)
	if err != nil {
		return nil, 0, err
	}
	liked, err := likedIDs(s.db, viewerID, model.TargetComment, ids)
	if err != nil {
		return nil, 0, err
	}

	out := make([]CommentView, 0, len(rows))
	for _, c := range rows {
		view := CommentView{
			ID:          c.ID,
			TargetType:  c.TargetType,
			TargetID:    c.TargetID,
			ReplyToID:   c.ReplyToID,
			ReplyToName: c.ReplyToName,
			Content:     c.Content,
			LikeCount:   c.LikeCount,
			Liked:       liked[c.ID],
			CanDelete:   viewerID != 0 && c.UserID == viewerID,
			CreatedAt:   c.CreatedAt,
		}
		if u, ok := users[c.UserID]; ok {
			view.User = briefOf(u)
		} else {
			view.User = &UserBrief{ID: c.UserID, Nickname: "佚名"}
		}
		out = append(out, view)
	}
	return out, total, nil
}

// CommentRequest 是发表评论的入参。
type CommentRequest struct {
	TargetType string `json:"targetType"`
	TargetID   uint64 `json:"targetId"`
	Content    string `json:"content"`
	ReplyToID  uint64 `json:"replyToId"`
}

// CreateComment 发表评论或回复。
func (s *InteractionService) CreateComment(userID uint64, req CommentRequest) (*CommentView, error) {
	if userID == 0 {
		return nil, forbidden("请先登录")
	}
	if !isCommentable(req.TargetType) {
		return nil, badRequest("该对象不支持评论")
	}
	n, err := targetCount(s.db, req.TargetType, req.TargetID)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, notFound("评论对象不存在")
	}
	content := strings.TrimSpace(req.Content)
	if rc := utf8.RuneCountInString(content); rc < 1 || rc > 500 {
		return nil, badRequest("评论需 1–500 字")
	}

	replyName := ""
	if req.ReplyToID != 0 {
		var parent model.Comment
		if err := s.db.First(&parent, req.ReplyToID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, badRequest("回复的评论不存在")
			}
			return nil, err
		}
		if parent.TargetType != req.TargetType || parent.TargetID != req.TargetID {
			return nil, badRequest("不能跨条目回复")
		}
		var pu model.User
		if err := s.db.Select("id", "nickname").First(&pu, parent.UserID).Error; err == nil {
			replyName = pu.Nickname
		}
	}

	c := model.Comment{
		TargetType:  req.TargetType,
		TargetID:    req.TargetID,
		UserID:      userID,
		ReplyToID:   req.ReplyToID,
		ReplyToName: replyName,
		Content:     content,
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&c).Error; err != nil {
			return err
		}
		return adjustTargetCount(tx, req.TargetType, req.TargetID, "comment_count", 1)
	})
	if err != nil {
		return nil, err
	}

	var u model.User
	_ = s.db.Select("id", "username", "nickname").First(&u, userID).Error
	return &CommentView{
		ID:          c.ID,
		TargetType:  c.TargetType,
		TargetID:    c.TargetID,
		ReplyToID:   c.ReplyToID,
		ReplyToName: c.ReplyToName,
		Content:     c.Content,
		User:        briefOf(u),
		CanDelete:   true,
		CreatedAt:   c.CreatedAt,
	}, nil
}

// DeleteComment 删除自己的评论。
func (s *InteractionService) DeleteComment(userID, id uint64) error {
	var c model.Comment
	if err := s.db.First(&c, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return notFound("评论不存在")
		}
		return err
	}
	if userID == 0 || c.UserID != userID {
		return forbidden("只能删除自己的评论")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := deleteTargets(tx, model.TargetComment, []uint64{c.ID}); err != nil {
			return err
		}
		if err := tx.Delete(&model.Comment{}, c.ID).Error; err != nil {
			return err
		}
		return adjustTargetCount(tx, c.TargetType, c.TargetID, "comment_count", -1)
	})
}

// deleteTargets 删除挂在对象上的点赞与评论。删除史枝 / 条目时一并清理，
// 避免留下永远读不到的孤儿行。
func deleteTargets(tx *gorm.DB, targetType string, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	var commentIDs []uint64
	if err := tx.Model(&model.Comment{}).
		Where("target_type = ? AND target_id IN ?", targetType, ids).
		Pluck("id", &commentIDs).Error; err != nil {
		return err
	}
	if len(commentIDs) > 0 {
		if err := tx.Where("target_type = ? AND target_id IN ?", model.TargetComment, commentIDs).
			Delete(&model.Like{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("target_type = ? AND target_id IN ?", targetType, ids).
		Delete(&model.Like{}).Error; err != nil {
		return err
	}
	return tx.Where("target_type = ? AND target_id IN ?", targetType, ids).
		Delete(&model.Comment{}).Error
}
