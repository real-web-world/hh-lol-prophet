package lcu

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	"github.com/real-web-world/hh-lol-prophet/services/lcu/models"
)

const (
	AuthUserName = "riot"
	Host         = "127.0.0.1"
	HttpScheme   = "https"
	WsScheme     = "wss"
)

var (
	ErrLolProcessNotFound = errors.New("未找到lol进程")
)

func GetLolClientApiInfo() (int, string, error) {
	return GetLolClientApiInfoAdapt()

}

func ConvertCurrSummonerToSummoner(currSummoner *models.SummonerProfileData) *models.Summoner {
	summonerLevel, _ := strconv.Atoi(currSummoner.Lol.Level)
	return &models.Summoner{
		AccountId:                   currSummoner.SummonerId,
		GameName:                    currSummoner.GameName,
		TagLine:                     currSummoner.GameTag,
		DisplayName:                 currSummoner.Name,
		InternalName:                currSummoner.Name,
		NameChangeFlag:              false,
		PercentCompleteForNextLevel: 0,
		ProfileIconId:               currSummoner.Icon,
		Puuid:                       currSummoner.Puuid,
		//RerollPoints:  ,
		SummonerId:       currSummoner.SummonerId,
		SummonerLevel:    summonerLevel,
		Unnamed:          false,
		XpSinceLastLevel: 0,
		XpUntilNextLevel: 0,
	}
}

func GenerateClientApiUrl(port int, token string) string {
	return fmt.Sprintf("%s://%s:%s@%s:%d", HttpScheme, AuthUserName, token, Host, port)
}
func GenerateClientWsUrl(port int) string {
	return fmt.Sprintf("%s://%s:%d", WsScheme, Host, port)
}
