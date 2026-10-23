package handler

import (
	"gonghe/internal/response"
	"gonghe/internal/service"

	"github.com/gin-gonic/gin"
)

// ListBranches 是「史枝总目」。
func (h *Handler) ListBranches(c *gin.Context) {
	page, size := h.page(c)
	f := service.BranchFilter{
		Scope:    c.DefaultQuery("scope", "all"),
		Sort:     c.DefaultQuery("sort", "hot"),
		Q:        c.Query("q"),
		Page:     page,
		PageSize: size,
	}
	list, total, err := h.Branches.List(userID(c), f)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, response.NewPage(list, total, page, size))
}

// DetailBranch 含祖先链、子枝与同门。
func (h *Handler) DetailBranch(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	detail, err := h.Branches.Detail(userID(c), id)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, detail)
}

// BranchTimeline 是某支的完整叙事：承继条目 + 自有条目，按年排序。
func (h *Handler) BranchTimeline(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	page, size := h.page(c)
	list, total, err := h.Branches.Timeline(userID(c), id, page, size, c.Query("own") == "1")
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, response.NewPage(list, total, page, size))
}

type forkRequest struct {
	ParentID    uint64 `json:"parentId"`
	ForkEventID uint64 `json:"forkEventId"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CreateBranch 从某一支的某一处分叉，另立新史。
func (h *Handler) CreateBranch(c *gin.Context) {
	var req forkRequest
	if !bind(c, &req) {
		return
	}
	detail, err := h.Branches.Fork(userID(c), service.ForkRequest{
		ParentID:    req.ParentID,
		ForkEventID: req.ForkEventID,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		fail(c, err)
		return
	}
	response.Created(c, detail)
}

type updateBranchRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *Handler) UpdateBranch(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req updateBranchRequest
	if !bind(c, &req) {
		return
	}
	if err := h.Branches.Update(userID(c), id, req.Name, req.Description); err != nil {
		fail(c, err)
		return
	}
	detail, err := h.Branches.Detail(userID(c), id)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, detail)
}

func (h *Handler) DeleteBranch(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.Branches.Delete(userID(c), id); err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": id})
}

// Graph 返回谱系图的全部节点。
func (h *Handler) Graph(c *gin.Context) {
	nodes, err := h.Branches.Graph()
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, nodes)
}

// Stats 是报头的大数字。
func (h *Handler) Stats(c *gin.Context) {
	stats, err := h.Branches.Stats()
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, stats)
}

// MyBranches 是我立的枝。
func (h *Handler) MyBranches(c *gin.Context) {
	page, size := h.page(c)
	list, total, err := h.Branches.List(userID(c), service.BranchFilter{
		Scope:    "all",
		Sort:     "new",
		OwnerID:  userID(c),
		Page:     page,
		PageSize: size,
	})
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, response.NewPage(list, total, page, size))
}
