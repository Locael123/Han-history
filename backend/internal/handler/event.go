package handler

import (
	"gonghe/internal/response"
	"gonghe/internal/service"

	"github.com/gin-gonic/gin"
)

// ListEvents 是全站条目流：所有史枝的实录混在一起，按时间或热度。
func (h *Handler) ListEvents(c *gin.Context) {
	page, size := h.page(c)
	f := service.EventFilter{
		Scope:    c.DefaultQuery("scope", "all"),
		Sort:     c.DefaultQuery("sort", "latest"),
		Q:        c.Query("q"),
		Page:     page,
		PageSize: size,
	}
	list, total, err := h.Events.Feed(userID(c), f)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, response.NewPage(list, total, page, size))
}

func (h *Handler) DetailEvent(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	detail, err := h.Events.Detail(userID(c), id)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, detail)
}

type eventRequest struct {
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Year     int      `json:"year"`
	Month    int      `json:"month"`
	Day      int      `json:"day"`
	Location string   `json:"location"`
	Figures  []string `json:"figures"`
	Tags     []string `json:"tags"`
}

func (r eventRequest) toService() service.EventRequest {
	return service.EventRequest{
		Title:    r.Title,
		Content:  r.Content,
		Year:     r.Year,
		Month:    r.Month,
		Day:      r.Day,
		Location: r.Location,
		Figures:  r.Figures,
		Tags:     r.Tags,
	}
}

// CreateEvent 给某支续写一条实录。
func (h *Handler) CreateEvent(c *gin.Context) {
	branchID, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req eventRequest
	if !bind(c, &req) {
		return
	}
	detail, err := h.Events.Create(userID(c), branchID, req.toService())
	if err != nil {
		fail(c, err)
		return
	}
	response.Created(c, detail)
}

func (h *Handler) UpdateEvent(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req eventRequest
	if !bind(c, &req) {
		return
	}
	detail, err := h.Events.Update(userID(c), id, req.toService())
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, detail)
}

func (h *Handler) DeleteEvent(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.Events.Delete(userID(c), id); err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": id})
}

func (h *Handler) MyEvents(c *gin.Context) {
	page, size := h.page(c)
	list, total, err := h.Events.Mine(userID(c), page, size)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, response.NewPage(list, total, page, size))
}
