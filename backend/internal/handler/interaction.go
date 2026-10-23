package handler

import (
	"gonghe/internal/response"
	"gonghe/internal/service"

	"github.com/gin-gonic/gin"
)

type likeRequest struct {
	TargetType string `json:"targetType"`
	TargetID   uint64 `json:"targetId"`
}

// ToggleLike 点赞 / 取消点赞。同一个接口，切换语义。
func (h *Handler) ToggleLike(c *gin.Context) {
	var req likeRequest
	if !bind(c, &req) {
		return
	}
	liked, count, err := h.Interactions.ToggleLike(userID(c), req.TargetType, req.TargetID)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, service.LikeResult{Liked: liked, Count: count})
}

// Comments 是一个对象（史枝 / 条目）下的评论。
func (h *Handler) Comments(c *gin.Context) {
	targetType := c.Query("targetType")
	targetID := uint64(queryInt(c, "targetId", 0, 0))
	if targetID == 0 {
		response.BadRequest(c, "缺少评论对象")
		return
	}
	page, size := h.page(c)
	list, total, err := h.Interactions.Comments(userID(c), targetType, targetID, page, size)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, response.NewPage(list, total, page, size))
}

type commentRequest struct {
	TargetType string `json:"targetType"`
	TargetID   uint64 `json:"targetId"`
	Content    string `json:"content"`
	ReplyToID  uint64 `json:"replyToId"`
}

func (h *Handler) CreateComment(c *gin.Context) {
	var req commentRequest
	if !bind(c, &req) {
		return
	}
	view, err := h.Interactions.CreateComment(userID(c), service.CommentRequest{
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		Content:    req.Content,
		ReplyToID:  req.ReplyToID,
	})
	if err != nil {
		fail(c, err)
		return
	}
	response.Created(c, view)
}

func (h *Handler) DeleteComment(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.Interactions.DeleteComment(userID(c), id); err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": id})
}
