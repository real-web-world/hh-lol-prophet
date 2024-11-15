package lcu

import (
	"github.com/pkg/errors"
	"github.com/real-web-world/hh-lol-prophet/services/lcu/models"
)

var (
	ErrLolProcessNotFound = errors.New("未找到lol进程")
)

func GetLolClientApiInfo() (int, string, error) {
	return GetLolClientApiInfoAdapt()
}

func ConvertCurrSummonerToSummoner(currSummoner *models.CurrSummoner) *models.Summoner {
	return &models.Summoner{
		AccountId:                   currSummoner.AccountId,
		GameName:                    currSummoner.GameName,
		TagLine:                     currSummoner.TagLine,
		DisplayName:                 currSummoner.DisplayName,
		InternalName:                currSummoner.InternalName,
		NameChangeFlag:              currSummoner.NameChangeFlag,
		PercentCompleteForNextLevel: currSummoner.PercentCompleteForNextLevel,
		ProfileIconId:               currSummoner.ProfileIconId,
		Puuid:                       currSummoner.Puuid,
		RerollPoints:                currSummoner.RerollPoints,
		SummonerId:                  currSummoner.SummonerId,
		SummonerLevel:               currSummoner.SummonerLevel,
		Unnamed:                     currSummoner.Unnamed,
		XpSinceLastLevel:            currSummoner.XpSinceLastLevel,
		XpUntilNextLevel:            currSummoner.XpUntilNextLevel,
	}
}
