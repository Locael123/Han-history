package handler

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"gonghe/internal/auth"
	"gonghe/internal/model"
	"gonghe/internal/response"
	"gonghe/internal/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResult struct {
	Token string            `json:"token"`
	User  *service.UserBrief `json:"user"`
}

// MeView 是「我的」页需要的账号信息与小统计。
type MeView struct {
	ID          uint64            `json:"id"`
	Username    string            `json:"username"`
	Nickname    string            `json:"nickname"`
	Bio         string            `json:"bio"`
	CreatedAt   time.Time         `json:"createdAt"`
	BranchCount int64             `json:"branchCount"`
	EventCount  int64             `json:"eventCount"`
}

func brief(u model.User) *service.UserBrief {
	return &service.UserBrief{ID: u.ID, Username: u.Username, Nickname: u.Nickname}
}

// Register 开放注册。用户名即登录名，昵称可中文。
func (h *Handler) Register(c *gin.Context) {
	if !h.Cfg.AllowRegister {
		response.Forbidden(c, "本站暂未开放注册")
		return
	}
	var req registerRequest
	if !bind(c, &req) {
		return
	}
	username := strings.TrimSpace(req.Username)
	nickname := strings.TrimSpace(req.Nickname)
	if !usernameRe.MatchString(username) {
		response.BadRequest(c, "名号需为 3–20 位字母、数字或下划线")
		return
	}
	if len(req.Password) < 6 || len(req.Password) > 64 {
		response.BadRequest(c, "口令长度需在 6–64 位之间")
		return
	}
	if nickname == "" {
		nickname = username
	}
	if n := len([]rune(nickname)); n > 20 {
		response.BadRequest(c, "雅号不能超过 20 个字")
		return
	}

	var count int64
	if err := h.DB.Model(&model.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		fail(c, err)
		return
	}
	if count > 0 {
		response.Conflict(c, "这个名号已经有人用了")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(c, err)
		return
	}
	u := model.User{Username: username, Nickname: nickname, PasswordHash: string(hash)}
	if err := h.DB.Create(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			response.Conflict(c, "这个名号已经有人用了")
			return
		}
		fail(c, err)
		return
	}
	token, err := auth.Sign(h.Cfg, u.ID, u.Username)
	if err != nil {
		fail(c, err)
		return
	}
	response.Created(c, authResult{Token: token, User: brief(u)})
}

// Login 用名号 + 口令换令牌。
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if !bind(c, &req) {
		return
	}
	username := strings.TrimSpace(req.Username)
	var u model.User
	err := h.DB.Where("username = ?", username).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Fail(c, 401, response.CodeUnauthorized, "名号或口令不对")
		return
	}
	if err != nil {
		fail(c, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		response.Fail(c, 401, response.CodeUnauthorized, "名号或口令不对")
		return
	}
	token, err := auth.Sign(h.Cfg, u.ID, u.Username)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, authResult{Token: token, User: brief(u)})
}

// Me 返回当前账号与我写下的东西的数量。
func (h *Handler) Me(c *gin.Context) {
	uid := userID(c)
	var u model.User
	if err := h.DB.First(&u, uid).Error; err != nil {
		fail(c, err)
		return
	}
	view := MeView{
		ID:        u.ID,
		Username:  u.Username,
		Nickname:  u.Nickname,
		Bio:       u.Bio,
		CreatedAt: u.CreatedAt,
	}
	h.DB.Model(&model.Branch{}).Where("owner_id = ?", uid).Count(&view.BranchCount)
	h.DB.Model(&model.Event{}).Where("user_id = ?", uid).Count(&view.EventCount)
	response.OK(c, view)
}
