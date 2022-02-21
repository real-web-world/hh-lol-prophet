package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	app "github.com/real-web-world/hh-lol-prophet"
	"github.com/real-web-world/hh-lol-prophet/bootstrap"
	"github.com/real-web-world/hh-lol-prophet/global"
	"github.com/real-web-world/hh-lol-prophet/pkg/bdk"
)

const (
	procName    = "hh-lol-prophet.exe"
	procNewName = "hh-lol-prophet_new.exe"
	updateApi   = "https://lol.buffge.com/latestApp.json"
)

type (
	AppUpdateInfo struct {
		Version     string `json:"version"`
		DownloadUrl string `json:"downloadUrl"`
	}
)

var (
	showVersion   = flag.Bool("v", false, "展示版本信息")
	isUpdate      = flag.Bool("u", false, "是否是更新")
	delUpgradeBin = flag.Bool("delUpgradeBin", false, "是否删除升级程序")
)

func flagInit() {
	flag.Parse()
	if *showVersion {
		log.Printf("当前版本:%s,commitID:%s,构建时间:%v\n", app.APPVersion,
			app.Commit, app.BuildTime)
		os.Exit(0)
		return
	}
	if *isUpdate {
		err := selfUpdate()
		if err != nil {
			log.Println("selfUpdate failed,", err)
		}
		return
	} else {
		_ = mustRunWithMain()
	}
	if *delUpgradeBin {
		go func() {
			_ = removeUpgradeBinFile()
		}()
	}
}

func mustRunWithMain() error {
	binPath, err := os.Executable()
	if err != nil {
		return err
	}
	binFileName := filepath.Base(binPath)
	if binFileName == procNewName {
		os.Exit(-1)
	}
	return nil
}
func main() {
	flagInit()
	if err := checkUpdate(); err != nil {
		log.Println("检查更新失败,", err)
	}
	err := bootstrap.InitApp()
	defer global.Cleanup()
	if err != nil {
		panic(fmt.Sprintf("初始化应用失败:%v\n", err))
	}
	prophet := app.NewProphet()
	if err = prophet.Run(); err != nil {
		log.Fatal(err)
	}
}
func removeUpgradeBinFile() error {
	binNewPath, err := os.Executable()
	if err != nil {
		return err
	}
	if filepath.Base(binNewPath) != procName {
		return errors.New("当前不是主进程 禁止执行")
	}
	dirPath, err := filepath.Abs(filepath.Dir(binNewPath))
	if err != nil {
		return err
	}
	binNewFullPath := filepath.Join(dirPath, procNewName)
	return os.Remove(binNewFullPath)
}
func checkUpdate() error {
	var binNewFullPath string
	// get api
	resp, err := http.Get(updateApi)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bts, _ := io.ReadAll(resp.Body)
	updateInfo := &AppUpdateInfo{}
	if json.Unmarshal(bts, updateInfo) != nil {
		return nil
	}
	if updateInfo.Version == "" || updateInfo.DownloadUrl == "" {
		return nil
	}
	if bdk.CompareVersion(updateInfo.Version, app.APPVersion) <= 0 {
		// log.Println("已是最新无需下载")
		return nil
	}
	log.Println("检测到更新,两秒后将更新或按回车立即更新")
	updateTimeoutC := time.After(time.Second * 2)
	done := make(chan struct{})
	go func() {
		str := ""
		_, _ = fmt.Scanln(&str)
		done <- struct{}{}
	}()
	// download new.exe
	{
		resp, err = http.Get(updateInfo.DownloadUrl)
		if err != nil {
			log.Println("下载最新二进制失败")
			return err
		}
		defer resp.Body.Close()
		binNewPath, err := os.Executable()
		if err != nil {
			return err
		}
		dirPath, err := filepath.Abs(filepath.Dir(binNewPath))
		if err != nil {
			return err
		}
		binNewFullPath = filepath.Join(dirPath, procNewName)
		binNewFile, err := os.OpenFile(binNewFullPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0664)
		if err != nil {
			return err
		}
		_, err = io.Copy(binNewFile, resp.Body)
		if err != nil {
			_ = binNewFile.Close()
			return errors.New("下载更新文件失败")
		}
		_ = binNewFile.Close()
	}
	select {
	case <-done:
	case <-updateTimeoutC:
	}
	// 输出信息 即将更新
	// 启动ttt_new.exe
	{
		cmd := exec.Command("cmd.exe", "/C", "start", binNewFullPath, "-u", "true")
		if err := cmd.Run(); err != nil {
			log.Println("启动更新进程失败:", err)
			return err
		}
		os.Exit(0)
	}
	return nil
}
func selfUpdate() error {
	binNewPath, err := os.Executable()
	if err != nil {
		return err
	}
	dirPath, err := filepath.Abs(filepath.Dir(binNewPath))
	if err != nil {
		return err
	}
	binNewFileName := filepath.Base(binNewPath)
	if binNewFileName != procNewName {
		return nil
	}
	binFullPath := filepath.Join(dirPath, procName)
	binNewFile, err := os.Open(binNewPath)
	if err != nil {
		defer binNewFile.Close()
	}
	if !bdk.IsFile(binFullPath) {
		return errors.New("二进制文件不存在")
	}
	binFile, err := os.OpenFile(binFullPath, os.O_WRONLY|os.O_TRUNC, 0664)
	if err != nil {
		log.Println("二进制文件被占用或不存在")
		return err
	}
	_, err = io.Copy(binFile, binNewFile)
	if err != nil {
		log.Println("写入二进制文件 更新失败")
		return err
	}
	_ = binFile.Close()
	cmd := exec.Command("cmd.exe", "/C", "start", binFullPath, "-delUpgradeBin", "true")
	if err = cmd.Run(); err != nil {
		log.Println("Error:", err)
		return err
	}
	os.Exit(0)
	return nil
}
