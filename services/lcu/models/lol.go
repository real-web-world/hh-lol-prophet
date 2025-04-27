package models

type (
	GameMode      string // 游戏模式
	GameQueueID   int    // 游戏队列模式id
	GameQueueType string // 游戏队列模式
	GameStatus    string // 游戏状态
	RankTier      string // 排位等级
	GameType      string // 游戏类型
	Spell         int    // 召唤师技能
	Champion      int    // 英雄
	Lane          string // 位置
	ChampionRole  string // 英雄角色
	GameFlow      string // 游戏状态
	MapID         int    // 地图id
	TeamID        int    // 队伍id
	TeamIDStr     string // 队伍id
	PlatformId    string // 大区id
)

const (
	GameWin  = "Win"
	GameFail = "Fail"
)

// 游戏模式
const (
	GameModeNone    GameMode = ""
	GameModeClassic GameMode = "CLASSIC"      // 经典模式
	GameModeARAM    GameMode = "ARAM"         // 大乱斗
	GameModeTFT     GameMode = "TFT"          // 云顶之弈
	GameModeURF     GameMode = "URF"          // 无限火力
	GameModeCustom  GameMode = "PRACTICETOOL" // 自定义
	GameModeCherry  GameMode = "CHERRY"       // 斗魂竞技场
)

// 队列模式
const (
	GameQueueTypeNormal   GameQueueType = "NORMAL"            // 匹配
	GameQueueTypeRankSolo GameQueueType = "RANKED_SOLO_5x5"   // 单双排
	GameQueueTypeRankFlex GameQueueType = "RANKED_FLEX_SR"    // 组排
	GameQueueTypeARAM     GameQueueType = "ARAM_UNRANKED_5x5" // 大乱斗5v5
	GameQueueTypeURF      GameQueueType = "URF"               // 无限火力
	GameQueueTypeBOT      GameQueueType = "BOT"               // 人机
	GameQueueTypeCustom   GameQueueType = "PRACTICETOOL"      // 自定义
)

// 游戏状态
const (
	GameStatusInQueue        GameStatus = "inQueue"                   // 队列中
	GameStatusInGame         GameStatus = "inGame"                    // 游戏中
	GameStatusChampionSelect GameStatus = "championSelect"            // 英雄选择中
	GameStatusOutOfGame      GameStatus = "outOfGame"                 // 退出游戏中
	GameStatusHostNormal     GameStatus = "hosting_NORMAL"            // 匹配组队中-队长
	GameStatusHostRankSolo   GameStatus = "hosting_RANKED_SOLO_5x5"   // 单排组队中-队长
	GameStatusHostRankFlex   GameStatus = "hosting_RANKED_FLEX_SR"    // 组排组队中-队长
	GameStatusHostARAM       GameStatus = "hosting_ARAM_UNRANKED_5x5" // 大乱斗5v5组队中-队长
	GameStatusHostURF        GameStatus = "hosting_URF"               // 无限火力组队中-队长
	GameStatusHostBOT        GameStatus = "hosting_BOT"               // 人机组队中-队长
)
const (
	GameFlowChampionSelect GameFlow = "ChampSelect" // 英雄选择中
	GameFlowReadyCheck     GameFlow = "ReadyCheck"  // 等待接受对局
	GameFlowInProgress     GameFlow = "InProgress"  // 进行中
	GameFlowMatchmaking    GameFlow = "Matchmaking" // 匹配中
	GameFlowNone           GameFlow = "None"        // 无
)

// 排位等级
const (
	RankTierIron        RankTier = "IRON"        // 黑铁
	RankTierBronze      RankTier = "BRONZE"      // 青铜
	RankTierSilver      RankTier = "SILVER"      // 白银
	RankTierGold        RankTier = "GOLD"        // 黄金
	RankTierPlatinum    RankTier = "PLATINUM"    // 白金
	RankTierDiamond     RankTier = "DIAMOND"     // 钻石
	RankTierMaster      RankTier = "MASTER"      // 大师
	RankTierGrandMaster RankTier = "GRANDMASTER" // 宗师
	RankTierChallenger  RankTier = "CHALLENGER"  // 王者
)

// 游戏类型
const (
	GameTypeMatch GameType = "MATCHED_GAME" // 匹配
)
const (
	// 游戏队列id
	NormalQueueID    GameQueueID = 430  // 匹配
	RankSoleQueueID  GameQueueID = 420  // 单排
	RankFlexQueueID  GameQueueID = 440  // 组排
	ARAMQueueID      GameQueueID = 450  // 大乱斗
	URFQueueID       GameQueueID = 900  // 无限火力
	BOTSimpleQueueID GameQueueID = 830  // 人机入门
	BOTNoviceQueueID GameQueueID = 840  // 人机新手
	BOTNormalQueueID GameQueueID = 850  // 人机一般
	CheeryQueueID    GameQueueID = 1700 // 斗魂竞技场
)

// 地图id
const (
	MapIDClassic MapID = 11 // 经典模式召唤师峡谷
	MapIDARAM    MapID = 12 // 极地大乱斗
	MapIDCheery  MapID = 30 // 斗魂竞技场
)

// 队伍id
const (
	TeamIDNone    TeamID    = 0     // 未知
	TeamIDBlue    TeamID    = 100   // 蓝色方
	TeamIDRed     TeamID    = 200   // 红色方
	TeamIDStrNone TeamIDStr = ""    // 未知
	TeamIDStrBlue TeamIDStr = "100" // 蓝色方
	TeamIDStrRed  TeamIDStr = "200" // 红色方
)

// 大区id
const (
	PlatformIDDX1 = "HN1" // 艾欧尼亚
	PlatformIDDX2 = "HN2" // 祖安
)

// 召唤师技能
const (
	SpellPingZhang Spell = 21 // 屏障
	SpellShanXian  Spell = 4  // 闪现
)

// 位置
const (
	LaneTop    Lane = "JUNGLE" // 上路
	LaneJungle Lane = "JUNGLE" // 打野
	LaneMiddle Lane = "MIDDLE" // 中路
	LaneBottom Lane = "BOTTOM" // 下路
)

// 英雄角色
const (
	ChampionRoleSolo    ChampionRole = "SOLE"        // 单人路
	ChampionRoleSupport ChampionRole = "DUO_SUPPORT" // 辅助
	ChampionRoleADC     ChampionRole = "DUO_CARRY"   // adc
	ChampionRoleNone    ChampionRole = "NONE"        // 无 一般是打野
)

// 游戏大区
const (
	PlatformIdHN1 PlatformId = "HN1"
)
