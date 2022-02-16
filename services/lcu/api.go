package lcu

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/real-web-world/hh-lol-prophet/services/lcu/models"

	"github.com/real-web-world/hh-lol-prophet/services/logger"
)

type (
	// 每单位的数据
	PerMinDeltas struct {
		Ten    float64 `json:"0-10"`
		Twenty float64 `json:"10-20"`
		Thirty float64 `json:"20-30"`
		Forty  float64 `json:"30-40"`
		Fifty  float64 `json:"40-50"`
		Sixty  float64 `json:"50-60"`
	}
)
type (
	ConversationMsgType string // 会话组消息类型
)

type (
	CommonResp struct {
		ErrorCode  string `json:"errorCode"`
		HttpStatus int    `json:"httpStatus"`
		Message    string `json:"message"`
	}
	CurrSummoner struct {
		AccountId                   int64  `json:"accountId"`
		DisplayName                 string `json:"displayName"`
		InternalName                string `json:"internalName"`
		NameChangeFlag              bool   `json:"nameChangeFlag"`
		PercentCompleteForNextLevel int    `json:"percentCompleteForNextLevel"`
		ProfileIconId               int    `json:"profileIconId"`
		Puuid                       string `json:"puuid"`
		RerollPoints                struct {
			CurrentPoints    int `json:"currentPoints"`
			MaxRolls         int `json:"maxRolls"`
			NumberOfRolls    int `json:"numberOfRolls"`
			PointsCostToRoll int `json:"pointsCostToRoll"`
			PointsToReroll   int `json:"pointsToReroll"`
		} `json:"rerollPoints"`
		SummonerId       int64 `json:"summonerId"`
		SummonerLevel    int   `json:"summonerLevel"`
		Unnamed          bool  `json:"unnamed"`
		XpSinceLastLevel int   `json:"xpSinceLastLevel"`
		XpUntilNextLevel int   `json:"xpUntilNextLevel"`
	}
	GameListResp struct {
		CommonResp
		AccountID int64    `json:"accountId"`
		Games     GameList `json:"games"`
	}
	GameList struct {
		GameBeginDate  string     `json:"gameBeginDate"`
		GameCount      int        `json:"gameCount"`
		GameEndDate    string     `json:"gameEndDate"`
		GameIndexBegin int        `json:"gameIndexBegin"`
		GameIndexEnd   int        `json:"gameIndexEnd"`
		Games          []GameInfo `json:"games"`
	}
	GameInfo struct {
		GameCreation          int64           `json:"gameCreation"` // 创建时间戳 ms
		GameCreationDate      time.Time       `json:"gameCreationDate"`
		GameDuration          int             `json:"gameDuration"` // 游戏持续时长 秒
		GameId                int64           `json:"gameId"`
		GameMode              models.GameMode `json:"gameMode"`
		GameType              models.GameType `json:"gameType"`
		GameVersion           string          `json:"gameVersion"`
		MapId                 int             `json:"mapId"` // 地图id
		ParticipantIdentities []struct {      // 参与者
			ParticipantId int      `json:"participantId"` // 参与者id
			Player        struct { // 玩家信息
				AccountId         int64  `json:"accountId"`         // 账号id
				CurrentAccountId  int64  `json:"currentAccountId"`  // 当前账号id
				CurrentPlatformId string `json:"currentPlatformId"` // 当前平台id
				MatchHistoryUri   string `json:"matchHistoryUri"`   // 匹配劣势url
				PlatformId        string `json:"platformId"`        // 平台id
				ProfileIcon       int    `json:"profileIcon"`       // 头像icon
				SummonerId        int64  `json:"summonerId"`        // 召唤师id
				SummonerName      string `json:"summonerName"`      // 召唤师名称
			} `json:"player"`
		} `json:"participantIdentities"`
		Participants []struct { // 参与者详细信息
			ChampionId                models.Champion `json:"championId"` // 英雄id
			HighestAchievedSeasonTier string          `json:"highestAchievedSeasonTier"`
			ParticipantId             int             `json:"participantId"`
			Spell1Id                  models.Spell    `json:"spell1Id"` // 召唤师技能1
			Spell2Id                  models.Spell    `json:"spell2Id"` // 召唤师技能2
			Stats                     struct {
				Assists                         int  `json:"assists"`                   // 助攻数
				CausedEarlySurrender            bool `json:"causedEarlySurrender"`      // 是否申请了提前投降
				ChampLevel                      int  `json:"champLevel"`                // 召唤师等级
				CombatPlayerScore               int  `json:"combatPlayerScore"`         //
				DamageDealtToObjectives         int  `json:"damageDealtToObjectives"`   // 对战略点的总伤害
				DamageDealtToTurrets            int  `json:"damageDealtToTurrets"`      // 对防御塔的总伤害
				DamageSelfMitigated             int  `json:"damageSelfMitigated"`       // 自我缓和的生命值
				Deaths                          int  `json:"deaths"`                    // 死亡次数
				DoubleKills                     int  `json:"doubleKills"`               // 双杀次数
				EarlySurrenderAccomplice        bool `json:"earlySurrenderAccomplice"`  // 是否同意了提前投降
				FirstBloodAssist                bool `json:"firstBloodAssist"`          // 是否助攻了一血
				FirstBloodKill                  bool `json:"firstBloodKill"`            // 是否获得了一血
				FirstInhibitorAssist            bool `json:"firstInhibitorAssist"`      // 是否助攻了摧毁第一个水晶
				FirstInhibitorKill              bool `json:"firstInhibitorKill"`        // 是否摧毁了摧毁第一个水晶
				FirstTowerAssist                bool `json:"firstTowerAssist"`          // 是否助攻了摧毁一塔
				FirstTowerKill                  bool `json:"firstTowerKill"`            // 是否摧毁了一塔
				GameEndedInEarlySurrender       bool `json:"gameEndedInEarlySurrender"` // 游戏是否由提前投降结束的
				GameEndedInSurrender            bool `json:"gameEndedInSurrender"`      // 游戏是由投降结束的
				GoldEarned                      int  `json:"goldEarned"`                // 金币获取
				GoldSpent                       int  `json:"goldSpent"`                 // 金币使用
				InhibitorKills                  int  `json:"inhibitorKills"`            // 摧毁水晶数
				Item0                           int  `json:"item0"`                     // 物品1
				Item1                           int  `json:"item1"`
				Item2                           int  `json:"item2"`
				Item3                           int  `json:"item3"`
				Item4                           int  `json:"item4"`
				Item5                           int  `json:"item5"`
				Item6                           int  `json:"item6"`
				KillingSprees                   int  `json:"killingSprees"`                   // 多杀
				Kills                           int  `json:"kills"`                           // 击杀
				LargestCriticalStrike           int  `json:"largestCriticalStrike"`           // 最大暴击伤害
				LargestKillingSpree             int  `json:"largestKillingSpree"`             // 最高连杀
				LargestMultiKill                int  `json:"largestMultiKill"`                // 多杀次数
				LongestTimeSpentLiving          int  `json:"longestTimeSpentLiving"`          // 最长存活时间
				MagicDamageDealt                int  `json:"magicDamageDealt"`                // 造成的魔法伤害
				MagicDamageDealtToChampions     int  `json:"magicDamageDealtToChampions"`     // 对英雄造成的魔法伤害
				MagicalDamageTaken              int  `json:"magicalDamageTaken"`              // 承受的魔法伤害
				NeutralMinionsKilled            int  `json:"neutralMinionsKilled"`            // 击杀野怪
				NeutralMinionsKilledEnemyJungle int  `json:"neutralMinionsKilledEnemyJungle"` // 击杀敌方野怪
				NeutralMinionsKilledTeamJungle  int  `json:"neutralMinionsKilledTeamJungle"`  // 击杀队伍野怪
				ObjectivePlayerScore            int  `json:"objectivePlayerScore"`            //
				ParticipantId                   int  `json:"participantId"`
				PentaKills                      int  `json:"pentaKills"`
				Perk0                           int  `json:"perk0"`
				Perk0Var1                       int  `json:"perk0Var1"`
				Perk0Var2                       int  `json:"perk0Var2"`
				Perk0Var3                       int  `json:"perk0Var3"`
				Perk1                           int  `json:"perk1"`
				Perk1Var1                       int  `json:"perk1Var1"`
				Perk1Var2                       int  `json:"perk1Var2"`
				Perk1Var3                       int  `json:"perk1Var3"`
				Perk2                           int  `json:"perk2"`
				Perk2Var1                       int  `json:"perk2Var1"`
				Perk2Var2                       int  `json:"perk2Var2"`
				Perk2Var3                       int  `json:"perk2Var3"`
				Perk3                           int  `json:"perk3"`
				Perk3Var1                       int  `json:"perk3Var1"`
				Perk3Var2                       int  `json:"perk3Var2"`
				Perk3Var3                       int  `json:"perk3Var3"`
				Perk4                           int  `json:"perk4"`
				Perk4Var1                       int  `json:"perk4Var1"`
				Perk4Var2                       int  `json:"perk4Var2"`
				Perk4Var3                       int  `json:"perk4Var3"`
				Perk5                           int  `json:"perk5"`
				Perk5Var1                       int  `json:"perk5Var1"`
				Perk5Var2                       int  `json:"perk5Var2"`
				Perk5Var3                       int  `json:"perk5Var3"`
				PerkPrimaryStyle                int  `json:"perkPrimaryStyle"`
				PerkSubStyle                    int  `json:"perkSubStyle"`
				PhysicalDamageDealt             int  `json:"physicalDamageDealt"`            // 造成的物理伤害
				PhysicalDamageDealtToChampions  int  `json:"physicalDamageDealtToChampions"` // 对英雄造成的物理伤害
				PhysicalDamageTaken             int  `json:"physicalDamageTaken"`            // 受到的物理伤害
				PlayerScore0                    int  `json:"playerScore0"`
				PlayerScore1                    int  `json:"playerScore1"`
				PlayerScore2                    int  `json:"playerScore2"`
				PlayerScore3                    int  `json:"playerScore3"`
				PlayerScore4                    int  `json:"playerScore4"`
				PlayerScore5                    int  `json:"playerScore5"`
				PlayerScore6                    int  `json:"playerScore6"`
				PlayerScore7                    int  `json:"playerScore7"`
				PlayerScore8                    int  `json:"playerScore8"`
				PlayerScore9                    int  `json:"playerScore9"`
				QuadraKills                     int  `json:"quadraKills"`            // 四杀次数
				SightWardsBoughtInGame          int  `json:"sightWardsBoughtInGame"` //
				TeamEarlySurrendered            bool `json:"teamEarlySurrendered"`   // 队伍是否提前投降
				TimeCCingOthers                 int  `json:"timeCCingOthers"`
				TotalDamageDealt                int  `json:"totalDamageDealt"`            // 造成的伤害总和
				TotalDamageDealtToChampions     int  `json:"totalDamageDealtToChampions"` // 对英雄造成的伤害总和
				TotalDamageTaken                int  `json:"totalDamageTaken"`            // 对防御塔造成的伤害总和
				TotalHeal                       int  `json:"totalHeal"`                   // 治疗伤害
				TotalMinionsKilled              int  `json:"totalMinionsKilled"`          // 击杀小兵数
				TotalPlayerScore                int  `json:"totalPlayerScore"`
				TotalScoreRank                  int  `json:"totalScoreRank"`
				TotalTimeCrowdControlDealt      int  `json:"totalTimeCrowdControlDealt"` // 总控制时长
				TotalUnitsHealed                int  `json:"totalUnitsHealed"`           //
				TripleKills                     int  `json:"tripleKills"`                // 三杀次数
				TrueDamageDealt                 int  `json:"trueDamageDealt"`            //  总真实伤害
				TrueDamageDealtToChampions      int  `json:"trueDamageDealtToChampions"` // 对英雄的总真实伤害
				TrueDamageTaken                 int  `json:"trueDamageTaken"`            // 对防御塔的真实伤害
				TurretKills                     int  `json:"turretKills"`                // 击杀防御塔
				UnrealKills                     int  `json:"unrealKills"`                // 摧毁水晶
				VisionScore                     int  `json:"visionScore"`                // 视野得分
				VisionWardsBoughtInGame         int  `json:"visionWardsBoughtInGame"`    // 购买控制守卫
				WardsKilled                     int  `json:"wardsKilled"`                // 击杀守卫
				WardsPlaced                     int  `json:"wardsPlaced"`                // 放置守卫
				Win                             bool `json:"win"`                        // 是否获胜
			} `json:"stats"`
			TeamId   int `json:"teamId"`
			Timeline struct {
				CreepsPerMinDeltas          PerMinDeltas `json:"creepsPerMinDeltas"` // 每单位(分钟)移动码数(估计是千码)
				CsDiffPerMinDeltas          PerMinDeltas `json:"csDiffPerMinDeltas"`
				DamageTakenDiffPerMinDeltas PerMinDeltas `json:"damageTakenDiffPerMinDeltas"` // 每单位受到伤害差距
				DamageTakenPerMinDeltas     PerMinDeltas `json:"damageTakenPerMinDeltas"`     // 每单位受到伤害
				GoldPerMinDeltas            PerMinDeltas `json:"goldPerMinDeltas"`            // 每单位获得金币
				Lane                        string       `json:"lane"`                        // 哪一路
				ParticipantId               int          `json:"participantId"`               // 参与者id
				Role                        string       `json:"role"`                        // 角色
				XpDiffPerMinDeltas          PerMinDeltas `json:"xpDiffPerMinDeltas"`          // 每单位经验差距
				XpPerMinDeltas              PerMinDeltas `json:"xpPerMinDeltas"`              // 每单位经验数
			} `json:"timeline"`
		} `json:"participants"`
		PlatformId string             `json:"platformId"` // 平台id
		QueueId    models.GameQueueID `json:"queueId"`    // 队列id
		SeasonId   int                `json:"seasonId"`
		Teams      []interface{}      `json:"teams"`
	}
	// 聊天组
	Conversation struct {
		GameName           string            `json:"gameName"`
		GameTag            string            `json:"gameTag"`
		Id                 string            `json:"id"`
		InviterId          string            `json:"inviterId"`
		IsMuted            bool              `json:"isMuted"`
		LastMessage        interface{}       `json:"lastMessage"`
		Name               string            `json:"name"`
		Password           string            `json:"password"`
		Pid                string            `json:"pid"`
		TargetRegion       string            `json:"targetRegion"`
		Type               models.GameStatus `json:"type"`
		UnreadMessageCount int               `json:"unreadMessageCount"`
	}
	ConversationMsg struct {
		Body           string              `json:"body"`
		FromId         string              `json:"fromId"`
		FromPid        string              `json:"fromPid"`
		FromSummonerId int64               `json:"fromSummonerId"`
		Id             string              `json:"id"`
		IsHistorical   bool                `json:"isHistorical"`
		Timestamp      time.Time           `json:"timestamp"`
		Type           ConversationMsgType `json:"type"`
	}
	Summoner struct {
		AccountId                   int64  `json:"accountId"`
		DisplayName                 string `json:"displayName"`
		InternalName                string `json:"internalName"`
		NameChangeFlag              bool   `json:"nameChangeFlag"`
		PercentCompleteForNextLevel int    `json:"percentCompleteForNextLevel"`
		Privacy                     string `json:"privacy"`
		ProfileIconId               int    `json:"profileIconId"`
		Puuid                       string `json:"puuid"`
		RerollPoints                struct {
			CurrentPoints    int `json:"currentPoints"`
			MaxRolls         int `json:"maxRolls"`
			NumberOfRolls    int `json:"numberOfRolls"`
			PointsCostToRoll int `json:"pointsCostToRoll"`
			PointsToReroll   int `json:"pointsToReroll"`
		} `json:"rerollPoints"`
		SummonerId       int64 `json:"summonerId"`
		SummonerLevel    int   `json:"summonerLevel"`
		Unnamed          bool  `json:"unnamed"`
		XpSinceLastLevel int   `json:"xpSinceLastLevel"`
		XpUntilNextLevel int   `json:"xpUntilNextLevel"`
	}
	Participant struct {
		ChampionId                int    `json:"championId"`
		HighestAchievedSeasonTier string `json:"highestAchievedSeasonTier"`
		ParticipantId             int    `json:"participantId"`
		Spell1Id                  int    `json:"spell1Id"`
		Spell2Id                  int    `json:"spell2Id"`
		Stats                     struct {
			Assists                         int  `json:"assists"`
			CausedEarlySurrender            bool `json:"causedEarlySurrender"`
			ChampLevel                      int  `json:"champLevel"`
			CombatPlayerScore               int  `json:"combatPlayerScore"`
			DamageDealtToObjectives         int  `json:"damageDealtToObjectives"`
			DamageDealtToTurrets            int  `json:"damageDealtToTurrets"`
			DamageSelfMitigated             int  `json:"damageSelfMitigated"`
			Deaths                          int  `json:"deaths"`
			DoubleKills                     int  `json:"doubleKills"`
			EarlySurrenderAccomplice        bool `json:"earlySurrenderAccomplice"`
			FirstBloodAssist                bool `json:"firstBloodAssist"`
			FirstBloodKill                  bool `json:"firstBloodKill"`
			FirstInhibitorAssist            bool `json:"firstInhibitorAssist"`
			FirstInhibitorKill              bool `json:"firstInhibitorKill"`
			FirstTowerAssist                bool `json:"firstTowerAssist"`
			FirstTowerKill                  bool `json:"firstTowerKill"`
			GameEndedInEarlySurrender       bool `json:"gameEndedInEarlySurrender"`
			GameEndedInSurrender            bool `json:"gameEndedInSurrender"`
			GoldEarned                      int  `json:"goldEarned"`
			GoldSpent                       int  `json:"goldSpent"`
			InhibitorKills                  int  `json:"inhibitorKills"`
			Item0                           int  `json:"item0"`
			Item1                           int  `json:"item1"`
			Item2                           int  `json:"item2"`
			Item3                           int  `json:"item3"`
			Item4                           int  `json:"item4"`
			Item5                           int  `json:"item5"`
			Item6                           int  `json:"item6"`
			KillingSprees                   int  `json:"killingSprees"`
			Kills                           int  `json:"kills"`
			LargestCriticalStrike           int  `json:"largestCriticalStrike"`
			LargestKillingSpree             int  `json:"largestKillingSpree"`
			LargestMultiKill                int  `json:"largestMultiKill"`
			LongestTimeSpentLiving          int  `json:"longestTimeSpentLiving"`
			MagicDamageDealt                int  `json:"magicDamageDealt"`
			MagicDamageDealtToChampions     int  `json:"magicDamageDealtToChampions"`
			MagicalDamageTaken              int  `json:"magicalDamageTaken"`
			NeutralMinionsKilled            int  `json:"neutralMinionsKilled"`
			NeutralMinionsKilledEnemyJungle int  `json:"neutralMinionsKilledEnemyJungle"`
			NeutralMinionsKilledTeamJungle  int  `json:"neutralMinionsKilledTeamJungle"`
			ObjectivePlayerScore            int  `json:"objectivePlayerScore"`
			ParticipantId                   int  `json:"participantId"`
			PentaKills                      int  `json:"pentaKills"`
			Perk0                           int  `json:"perk0"`
			Perk0Var1                       int  `json:"perk0Var1"`
			Perk0Var2                       int  `json:"perk0Var2"`
			Perk0Var3                       int  `json:"perk0Var3"`
			Perk1                           int  `json:"perk1"`
			Perk1Var1                       int  `json:"perk1Var1"`
			Perk1Var2                       int  `json:"perk1Var2"`
			Perk1Var3                       int  `json:"perk1Var3"`
			Perk2                           int  `json:"perk2"`
			Perk2Var1                       int  `json:"perk2Var1"`
			Perk2Var2                       int  `json:"perk2Var2"`
			Perk2Var3                       int  `json:"perk2Var3"`
			Perk3                           int  `json:"perk3"`
			Perk3Var1                       int  `json:"perk3Var1"`
			Perk3Var2                       int  `json:"perk3Var2"`
			Perk3Var3                       int  `json:"perk3Var3"`
			Perk4                           int  `json:"perk4"`
			Perk4Var1                       int  `json:"perk4Var1"`
			Perk4Var2                       int  `json:"perk4Var2"`
			Perk4Var3                       int  `json:"perk4Var3"`
			Perk5                           int  `json:"perk5"`
			Perk5Var1                       int  `json:"perk5Var1"`
			Perk5Var2                       int  `json:"perk5Var2"`
			Perk5Var3                       int  `json:"perk5Var3"`
			PerkPrimaryStyle                int  `json:"perkPrimaryStyle"`
			PerkSubStyle                    int  `json:"perkSubStyle"`
			PhysicalDamageDealt             int  `json:"physicalDamageDealt"`
			PhysicalDamageDealtToChampions  int  `json:"physicalDamageDealtToChampions"`
			PhysicalDamageTaken             int  `json:"physicalDamageTaken"`
			PlayerScore0                    int  `json:"playerScore0"`
			PlayerScore1                    int  `json:"playerScore1"`
			PlayerScore2                    int  `json:"playerScore2"`
			PlayerScore3                    int  `json:"playerScore3"`
			PlayerScore4                    int  `json:"playerScore4"`
			PlayerScore5                    int  `json:"playerScore5"`
			PlayerScore6                    int  `json:"playerScore6"`
			PlayerScore7                    int  `json:"playerScore7"`
			PlayerScore8                    int  `json:"playerScore8"`
			PlayerScore9                    int  `json:"playerScore9"`
			QuadraKills                     int  `json:"quadraKills"`
			SightWardsBoughtInGame          int  `json:"sightWardsBoughtInGame"`
			TeamEarlySurrendered            bool `json:"teamEarlySurrendered"`
			TimeCCingOthers                 int  `json:"timeCCingOthers"`
			TotalDamageDealt                int  `json:"totalDamageDealt"`
			TotalDamageDealtToChampions     int  `json:"totalDamageDealtToChampions"`
			TotalDamageTaken                int  `json:"totalDamageTaken"`
			TotalHeal                       int  `json:"totalHeal"`
			TotalMinionsKilled              int  `json:"totalMinionsKilled"`
			TotalPlayerScore                int  `json:"totalPlayerScore"`
			TotalScoreRank                  int  `json:"totalScoreRank"`
			TotalTimeCrowdControlDealt      int  `json:"totalTimeCrowdControlDealt"`
			TotalUnitsHealed                int  `json:"totalUnitsHealed"`
			TripleKills                     int  `json:"tripleKills"`
			TrueDamageDealt                 int  `json:"trueDamageDealt"`
			TrueDamageDealtToChampions      int  `json:"trueDamageDealtToChampions"`
			TrueDamageTaken                 int  `json:"trueDamageTaken"`
			TurretKills                     int  `json:"turretKills"`
			UnrealKills                     int  `json:"unrealKills"`
			VisionScore                     int  `json:"visionScore"`
			VisionWardsBoughtInGame         int  `json:"visionWardsBoughtInGame"`
			WardsKilled                     int  `json:"wardsKilled"`
			WardsPlaced                     int  `json:"wardsPlaced"`
			Win                             bool `json:"win"`
		} `json:"stats"`
		TeamId   models.TeamID `json:"teamId"`
		Timeline struct {
			CreepsPerMinDeltas struct {
				Field1 float64 `json:"0-10"`
				Field2 float64 `json:"10-20"`
			} `json:"creepsPerMinDeltas"`
			CsDiffPerMinDeltas struct {
				Field1 float64 `json:"0-10"`
				Field2 float64 `json:"10-20"`
			} `json:"csDiffPerMinDeltas"`
			DamageTakenDiffPerMinDeltas struct {
				Field1 float64 `json:"0-10"`
				Field2 float64 `json:"10-20"`
			} `json:"damageTakenDiffPerMinDeltas"`
			DamageTakenPerMinDeltas struct {
				Field1 float64 `json:"0-10"`
				Field2 float64 `json:"10-20"`
			} `json:"damageTakenPerMinDeltas"`
			GoldPerMinDeltas struct {
				Field1 float64 `json:"0-10"`
				Field2 float64 `json:"10-20"`
			} `json:"goldPerMinDeltas"`
			Lane               models.Lane         `json:"lane"`
			ParticipantId      int                 `json:"participantId"`
			Role               models.ChampionRole `json:"role"`
			XpDiffPerMinDeltas struct {
				Field1 float64 `json:"0-10"`
				Field2 float64 `json:"10-20"`
			} `json:"xpDiffPerMinDeltas"`
			XpPerMinDeltas struct {
				Field1 float64 `json:"0-10"`
				Field2 float64 `json:"10-20"`
			} `json:"xpPerMinDeltas"`
		} `json:"timeline"`
	}
	GameSummary struct {
		GameCreation          int64           `json:"gameCreation"`
		GameCreationDate      time.Time       `json:"gameCreationDate"`
		GameDuration          int             `json:"gameDuration"`
		GameId                int64           `json:"gameId"`
		GameMode              models.GameMode `json:"gameMode"`
		GameType              models.GameType `json:"gameType"`
		GameVersion           string          `json:"gameVersion"`
		MapId                 models.MapID    `json:"mapId"`
		ParticipantIdentities []struct {
			ParticipantId int `json:"participantId"`
			Player        struct {
				AccountId         int64  `json:"accountId"`
				CurrentAccountId  int64  `json:"currentAccountId"`
				CurrentPlatformId string `json:"currentPlatformId"`
				MatchHistoryUri   string `json:"matchHistoryUri"`
				PlatformId        string `json:"platformId"`
				ProfileIcon       int    `json:"profileIcon"`
				SummonerId        int64  `json:"summonerId"`
				SummonerName      string `json:"summonerName"`
			} `json:"player"`
		} `json:"participantIdentities"`
		Participants []Participant `json:"participants"`
		PlatformId   string        `json:"platformId"`
		QueueId      int           `json:"queueId"`
		SeasonId     int           `json:"seasonId"`
		Teams        []struct {
			Bans []struct {
				ChampionId int `json:"championId"`
				PickTurn   int `json:"pickTurn"`
			} `json:"bans"`
			BaronKills           int    `json:"baronKills"`
			DominionVictoryScore int    `json:"dominionVictoryScore"`
			DragonKills          int    `json:"dragonKills"`
			FirstBaron           bool   `json:"firstBaron"`
			FirstBlood           bool   `json:"firstBlood"`
			FirstDargon          bool   `json:"firstDargon"`
			FirstInhibitor       bool   `json:"firstInhibitor"`
			FirstTower           bool   `json:"firstTower"`
			InhibitorKills       int    `json:"inhibitorKills"`
			RiftHeraldKills      int    `json:"riftHeraldKills"`
			TeamId               int    `json:"teamId"`
			TowerKills           int    `json:"towerKills"`
			VilemawKills         int    `json:"vilemawKills"`
			Win                  string `json:"win"`
		} `json:"teams"`
	}
)

const (
	JoinedRoomMsg                                 = "joined_room"
	ConversationMsgTypeSystem ConversationMsgType = "system"
)

// 获取当前召唤师
func GetCurrSummoner() (*CurrSummoner, error) {
	bts, err := cli.httpGet("/lol-summoner/v1/current-summoner")
	if err != nil {
		return nil, err
	}
	data := &CurrSummoner{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		logger.Info("获取当前召唤师失败", zap.Error(err))
		return nil, err
	}
	return data, nil
}

// 获取比赛记录
func ListGamesBySummonerID(summonerID int64, begin, limit int) (*GameListResp, error) {
	bts, err := cli.httpGet(fmt.Sprintf("/lol-match-history/v3/matchlist/account/%d?begIndex=%d&endIndex=%d",
		summonerID, begin, begin+limit))
	if err != nil {
		return nil, err
	}
	data := &GameListResp{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		logger.Info("获取比赛记录", zap.Error(err))
		return nil, err
	}
	return data, nil
}

// 获取会话组消息记录
func ListConversationMsg(conversationID string) ([]ConversationMsg, error) {
	bts, err := cli.httpGet(fmt.Sprintf("/lol-chat/v1/conversations/%s/messages", conversationID))
	if err != nil {
		return nil, err
	}
	list := make([]ConversationMsg, 0, 10)
	err = json.Unmarshal(bts, &list)
	if err != nil {
		logger.Info("获取会话组消息记录失败", zap.Error(err))
		return nil, err
	}
	return list, nil
}

// 获取当前对局聊天组
func GetCurrConversationID() (string, error) {
	bts, err := cli.httpGet("/lol-chat/v1/conversations")
	if err != nil {
		return "", err
	}
	list := make([]Conversation, 0, 1)
	err = json.Unmarshal(bts, &list)
	if err != nil {
		logger.Info("获取当前对局聊天组失败", zap.Error(err))
		return "", err
	}
	for _, conversation := range list {
		if conversation.Type == models.GameStatusChampionSelect {
			return conversation.Id, nil
		}
	}
	return "", errors.New("当前不在英雄选择阶段")
}

// 发送消息到聊天组
func SendConversationMsg(msg string, conversationID string) error {
	data := struct {
		Body string `json:"body"`
		Type string `json:"type"`
	}{
		Body: msg,
		Type: "chat",
	}
	_, err := cli.httpPost(fmt.Sprintf("/lol-chat/v1/conversations/%s/messages", conversationID), data)
	return err
}

// 申请加好友
func ApplyFriend(summonerID int64) error {
	data := struct {
		ID string `json:"id"`
	}{
		ID: strconv.FormatInt(summonerID, 10),
	}
	_, err := cli.httpPost("/lol-chat/v1/friend-requests", data)
	return err
}

// 取消加好友
func CancelApplyFriend(summonerID int64) error {
	_, err := cli.httpDel(fmt.Sprintf("/lol-chat/v1/friend-requests/%d", summonerID))
	return err
}

// 查询用户信息
func ListSummoner(summonerIDList []int64) ([]Summoner, error) {
	idStrList := make([]string, 0, len(summonerIDList))
	for _, id := range summonerIDList {
		idStrList = append(idStrList, strconv.FormatInt(id, 10))
	}
	bts, err := cli.httpGet(fmt.Sprintf("/lol-summoner/v2/summoners?ids=[%s]",
		strings.Join(idStrList, ",")))
	if len(bts) > 0 && bts[0] == '[' {
		list := make([]Summoner, 0, len(summonerIDList))
		err = json.Unmarshal(bts, &list)
		if err != nil {
			logger.Info("查询用户信息失败", zap.Error(err))
			return nil, err
		}
		return list, err
	}
	data := &CommonResp{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		logger.Info("查询用户信息失败", zap.Error(err))
		return nil, err
	}
	return nil, errors.New(data.Message)
}

// 查询用户信息
func QuerySummoner(summonerID int64) (*Summoner, error) {
	list, err := ListSummoner([]int64{summonerID})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, errors.New("获取召唤师信息失败 list == 0")
	}
	return &list[0], nil
}

// 查询对局详情
func QueryGameSummary(gameID int64) (*GameSummary, error) {
	bts, err := cli.httpGet(fmt.Sprintf("/lol-match-history/v1/games/%d", gameID))
	if err != nil {
		return nil, err
	}
	data := &GameSummary{}
	err = json.Unmarshal(bts, data)
	if err != nil {
		logger.Info("查询对局详情失败", zap.Error(err))
		return nil, err
	}
	return data, nil
}
