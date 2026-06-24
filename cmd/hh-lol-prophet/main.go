package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	appProj "github.com/real-web-world/hh-lol-prophet"
	"github.com/real-web-world/hh-lol-prophet/internal/bootstrap"
	"github.com/real-web-world/hh-lol-prophet/internal/config"
	"github.com/real-web-world/hh-lol-prophet/pkg/bdk"
)

var (
	showVersion   = flag.Bool("v", false, "展示版本信息")
	isUpdate      = flag.Bool("u", false, "是否是更新")
	delUpgradeBin = flag.Bool("delUpgradeBin", false, "是否删除升级程序")
)

func flagInit() {
	flag.Parse()
	if *showVersion {
		log.Printf("当前版本:%s,commitID:%s,构建时间:%v\n", appProj.APPVersion,
			appProj.Commit, appProj.BuildTime)
		os.Exit(0)
		return
	}
	if *isUpdate {
		err := selfUpdate()
		if err != nil {
			log.Println("selfUpdate failed,", err)
		}
		return
	}
	_ = mustRunWithMain()
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
	if binFileName == config.ProcNewName {
		os.Exit(-1)
	}
	return nil
}

func main() {
	flagInit()
	app, err := bootstrap.InitApp()
	if err != nil {
		slog.Error("lol对局先知初始化失败", slog.Any("err", err))
		os.Exit(1)
	}
	if err = app.Run(); err != nil {
		slog.Error("lol对局先知关闭失败", slog.Any("err", err))
	}
}

func removeUpgradeBinFile() error {
	binNewPath, err := os.Executable()
	if err != nil {
		return err
	}
	if filepath.Base(binNewPath) != config.ProcName {
		return errors.New("当前不是主进程 禁止执行")
	}
	dirPath, err := filepath.Abs(filepath.Dir(binNewPath))
	if err != nil {
		return err
	}
	binNewFullPath := filepath.Join(dirPath, config.ProcNewName)
	return os.Remove(binNewFullPath)
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
	if binNewFileName != config.ProcNewName {
		return nil
	}
	binFullPath := filepath.Join(dirPath, config.ProcName)
	binNewFile, err := os.Open(binNewPath)
	if err != nil {
		slog.Error("更新失败 os.Open(binNewPath)", slog.Any("err", err))
		return errors.New("更新失败")
	}
	defer func() {
		_ = binNewFile.Close()
	}()
	if !bdk.IsFile(binFullPath) {
		return errors.New("二进制文件不存在")
	}
	binFile, err := os.OpenFile(binFullPath, os.O_WRONLY|os.O_TRUNC, os.ModePerm)
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
