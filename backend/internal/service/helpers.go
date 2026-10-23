package service

import (
	"strings"
	"unicode/utf8"

	"gonghe/internal/model"

	"gorm.io/gorm"
)

// systemUser 是官修史枝的署名。
func systemUser() *UserBrief {
	return &UserBrief{ID: 0, Username: "shiguan", Nickname: "史馆"}
}

func briefOf(u model.User) *UserBrief {
	return &UserBrief{ID: u.ID, Username: u.Username, Nickname: u.Nickname}
}

func clampPage(page, pageSize, def, max int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = def
	}
	if pageSize > max {
		pageSize = max
	}
	return page, pageSize
}

// splitList 把逗号分隔的字符串拆成去空白、去重的数组。
// 兼容中文逗号、顿号、分号 —— 用户不会按程序员的习惯输入。
func splitList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}
	replacer := strings.NewReplacer("，", ",", "、", ",", "；", ",", ";", ",", "\n", ",")
	parts := strings.Split(replacer.Replace(s), ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func joinList(items []string) string {
	cleaned := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(strings.Trim(item, "，、,;；"))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		cleaned = append(cleaned, item)
	}
	return strings.Join(cleaned, "，")
}

func excerpt(s string, limit int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	runes := []rune(s)
	return string(runes[:limit]) + "…"
}

func uniqueIDs(ids []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(ids))
	out := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func loadUsers(db *gorm.DB, ids []uint64) (map[uint64]model.User, error) {
	out := make(map[uint64]model.User)
	ids = uniqueIDs(ids)
	if len(ids) == 0 {
		return out, nil
	}
	var users []model.User
	if err := db.Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, u := range users {
		out[u.ID] = u
	}
	return out, nil
}

func loadBranches(db *gorm.DB, ids []uint64) (map[uint64]model.Branch, error) {
	out := make(map[uint64]model.Branch)
	ids = uniqueIDs(ids)
	if len(ids) == 0 {
		return out, nil
	}
	var rows []model.Branch
	if err := db.Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, b := range rows {
		out[b.ID] = b
	}
	return out, nil
}

func likedIDs(db *gorm.DB, userID uint64, targetType string, ids []uint64) (map[uint64]bool, error) {
	out := make(map[uint64]bool)
	if userID == 0 {
		return out, nil
	}
	ids = uniqueIDs(ids)
	if len(ids) == 0 {
		return out, nil
	}
	var likes []model.Like
	if err := db.Where("user_id = ? AND target_type = ? AND target_id IN ?", userID, targetType, ids).
		Find(&likes).Error; err != nil {
		return nil, err
	}
	for _, l := range likes {
		out[l.TargetID] = true
	}
	return out, nil
}

// getBranch 读一条史枝，查不到时返回统一的 404。
func getBranch(db *gorm.DB, id uint64) (*model.Branch, error) {
	var b model.Branch
	if err := db.First(&b, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, notFound("史枝不存在")
		}
		return nil, err
	}
	return &b, nil
}

// eventDateLabel 只给日志 / 导出用，接口返回结构化的年月日。
func eventDateLabel(year, month, day int) string {
	var sb strings.Builder
	if year > 0 {
		sb.WriteString(strings.TrimSpace(strings.Repeat(" ", 0)))
		sb.WriteString(itoa(year))
		sb.WriteString("年")
	}
	if month > 0 {
		sb.WriteString(itoa(month))
		sb.WriteString("月")
	}
	if day > 0 {
		sb.WriteString(itoa(day))
		sb.WriteString("日")
	}
	return sb.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
