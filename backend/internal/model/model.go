// Package model 定义数据库实体。
//
// 数据模型刻意贴着 git 的隐喻：
//   - Branch 是「史枝」，root 枝为官修正史，用户从任一节点分叉出野史；
//   - Event 是「条目」，相当于一次提交，只能追加，不能改写别人已分叉的历史；
//   - ForkEventID 记录从父枝的哪一条目分出去，即 git 里的 fork point。
package model

import "time"

// User 是史官账号。用户名即登录名。
type User struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string    `gorm:"size:32;uniqueIndex;not null" json:"username"`
	Nickname     string    `gorm:"size:32;not null" json:"nickname"`
	PasswordHash string    `gorm:"size:100;not null" json:"-"`
	Bio          string    `gorm:"size:200" json:"bio"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// 点赞 / 评论的对象类型。
const (
	TargetBranch  = "branch"
	TargetEvent   = "event"
	TargetComment = "comment"
)

// Branch 是一条史枝。
//
// 统计字段全部反规范化（event_count / fork_count / like_count …），
// 读取列表时不必做聚合查询；写入时在同一事务里增减。
type Branch struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string `gorm:"size:80;not null" json:"name"`
	Description string `gorm:"size:600" json:"description"`

	// OwnerID = 0 表示官修（由「史馆」维护），普通用户不可续写。
	OwnerID  uint64 `gorm:"index;not null;default:0" json:"ownerId"`
	Official bool   `gorm:"index;not null;default:false" json:"official"`

	// 分叉信息：父枝、分叉点条目、分叉点年份、从父枝继承的条目数。
	// ForkEventID = 0 且 InheritedCount > 0 表示「自分叉时父枝的 HEAD 起」。
	ParentID       uint64 `gorm:"index;not null;default:0" json:"parentId"`
	ForkEventID    uint64 `gorm:"not null;default:0" json:"forkEventId"`
	ForkYear       int    `gorm:"not null;default:0" json:"forkYear"`
	InheritedCount int    `gorm:"not null;default:0" json:"inheritedCount"`

	// 自有条目数、自有条目最早 / 最晚年份（谱系图用）。
	EventCount int `gorm:"not null;default:0" json:"eventCount"`
	FirstYear  int `gorm:"not null;default:0" json:"firstYear"`
	LastYear   int `gorm:"not null;default:0" json:"lastYear"`

	ForkCount    int `gorm:"not null;default:0" json:"forkCount"`
	LikeCount    int `gorm:"not null;default:0" json:"likeCount"`
	CommentCount int `gorm:"not null;default:0" json:"commentCount"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Event 是一条历史条目（一次「提交」）。
//
// 时间用年 / 月 / 日三个整数存：民国史的精度常常只到年或月，
// 用 DATE 会逼着调用方编造一个「1 日」，排序也不方便。
type Event struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	BranchID uint64 `gorm:"index;not null" json:"branchId"`
	UserID   uint64 `gorm:"index;not null" json:"userId"`

	Title   string `gorm:"size:160;not null" json:"title"`
	Content string `gorm:"type:text;not null" json:"content"`

	Year  int `gorm:"not null;default:0" json:"year"`
	Month int `gorm:"not null;default:0" json:"month"`
	Day   int `gorm:"not null;default:0" json:"day"`

	Location string `gorm:"size:80" json:"location"`
	// Figures / Tags 以「，」或「,」分隔存一行，读取时拆成数组。
	// 人物和标签都是自由文本，不做独立表 —— 野史本来就不该被枚举框死。
	Figures string `gorm:"size:240" json:"figures"`
	Tags    string `gorm:"size:240" json:"tags"`

	LikeCount    int `gorm:"not null;default:0" json:"likeCount"`
	CommentCount int `gorm:"not null;default:0" json:"commentCount"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Like 是通用点赞。唯一索引保证一个人对同一对象只能赞一次。
type Like struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint64    `gorm:"not null;uniqueIndex:idx_like_unique,priority:1" json:"userId"`
	TargetType string    `gorm:"size:16;not null;uniqueIndex:idx_like_unique,priority:2" json:"targetType"`
	TargetID   uint64    `gorm:"not null;uniqueIndex:idx_like_unique,priority:3" json:"targetId"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Comment 是挂在史枝 / 条目 / 评论上的短评。
// 回复只做一层：ReplyToName 存快照，避免父评论删除后回复失去上下文。
type Comment struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	TargetType string `gorm:"size:16;not null" json:"targetType"`
	TargetID   uint64 `gorm:"not null" json:"targetId"`
	UserID     uint64 `gorm:"not null" json:"userId"`

	ReplyToID   uint64 `gorm:"not null;default:0" json:"replyToId"`
	ReplyToName string `gorm:"size:32" json:"replyToName"`

	Content   string `gorm:"size:1000;not null" json:"content"`
	LikeCount int    `gorm:"not null;default:0" json:"likeCount"`

	CreatedAt time.Time `json:"createdAt"`
}
