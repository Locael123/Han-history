package service

import "time"

// UserBrief 是精简的用户信息，避免把密码哈希之类的东西带出去。
type UserBrief struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

// BranchCard 是史枝在列表 / 卡片里的形态。
type BranchCard struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Official    bool   `json:"official"`
	Owner       *UserBrief `json:"owner"`

	ParentID       uint64 `json:"parentId"`
	ParentName     string `json:"parentName"`
	ForkEventID    uint64 `json:"forkEventId"`
	ForkEventTitle string `json:"forkEventTitle"`
	ForkYear       int    `json:"forkYear"`
	InheritedCount int    `json:"inheritedCount"`
	OwnCount       int    `json:"ownCount"`
	// EventCount 是「承继 + 自有」的总条目数，前端只认这个。
	EventCount   int `json:"eventCount"`
	ForkCount    int `json:"forkCount"`
	LikeCount    int `json:"likeCount"`
	CommentCount int `json:"commentCount"`

	FirstYear int       `json:"firstYear"`
	LastYear  int       `json:"lastYear"`
	Liked     bool      `json:"liked"`
	CanWrite  bool      `json:"canWrite"`
	CreatedAt time.Time `json:"createdAt"`
}

// LineageItem 是谱系链上的一节。
type LineageItem struct {
	ID         uint64 `json:"id"`
	Name       string `json:"name"`
	Official   bool   `json:"official"`
	ForkYear   int    `json:"forkYear"`
	EventCount int    `json:"eventCount"`
}

// BranchDetail 是史枝详情页需要的一切。
type BranchDetail struct {
	BranchCard
	Ancestors []LineageItem `json:"ancestors"`
	Children  []BranchCard  `json:"children"`
	Siblings  []BranchCard  `json:"siblings"`
	IsOwner   bool          `json:"isOwner"`
}

// TimelineItem 是时间轴上的一条实录，也可用于全站条目流。
type TimelineItem struct {
	ID       uint64 `json:"id"`
	BranchID uint64 `json:"branchId"`
	Branch   string `json:"branchName"`
	Official bool   `json:"branchOfficial"`

	Title    string `json:"title"`
	Content  string `json:"content,omitempty"`
	Excerpt  string `json:"excerpt"`
	Year     int    `json:"year"`
	Month    int    `json:"month"`
	Day      int    `json:"day"`
	Location string `json:"location"`
	Figures  []string `json:"figures"`
	Tags     []string `json:"tags"`

	Author *UserBrief `json:"author"`
	LikeCount    int  `json:"likeCount"`
	CommentCount int  `json:"commentCount"`
	Liked        bool `json:"liked"`
	Inherited    bool `json:"inherited"`

	CreatedAt time.Time `json:"createdAt"`
}

// EventDetail 在条目详情里额外带上所属史枝与编辑权限。
type EventDetail struct {
	TimelineItem
	Branch  BranchCard `json:"branch"`
	CanEdit bool       `json:"canEdit"`
}

// CommentView 是评论列表项。
type CommentView struct {
	ID          uint64     `json:"id"`
	TargetType  string     `json:"targetType"`
	TargetID    uint64     `json:"targetId"`
	ReplyToID   uint64     `json:"replyToId"`
	ReplyToName string     `json:"replyToName"`
	Content     string     `json:"content"`
	User        *UserBrief `json:"user"`
	LikeCount   int        `json:"likeCount"`
	Liked       bool       `json:"liked"`
	CanDelete   bool       `json:"canDelete"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// GraphNode 是谱系图的一个节点。坐标交给前端算，服务端只给事实。
type GraphNode struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Official    bool   `json:"official"`
	ParentID    uint64 `json:"parentId"`
	ForkEventID uint64 `json:"forkEventId"`
	ForkYear    int    `json:"forkYear"`
	StartYear   int    `json:"startYear"`
	EndYear     int    `json:"endYear"`
	EventCount  int    `json:"eventCount"`
	ForkCount   int    `json:"forkCount"`
	LikeCount   int    `json:"likeCount"`
	Owner       string `json:"owner"`
}

// Stats 是首页报头的大数字。
type Stats struct {
	OfficialEvents   int64 `json:"officialEvents"`
	FolkEvents       int64 `json:"folkEvents"`
	TotalEvents      int64 `json:"totalEvents"`
	TotalBranches    int64 `json:"totalBranches"`
	OfficialBranches int64 `json:"officialBranches"`
	FolkBranches     int64 `json:"folkBranches"`
	Chroniclers    int64 `json:"chroniclers"`
	TotalLikes     int64 `json:"totalLikes"`
	TodayEvents    int64 `json:"todayEvents"`
}

// LikeResult 是点赞切换后的状态。
type LikeResult struct {
	Liked bool `json:"liked"`
	Count int  `json:"count"`
}
