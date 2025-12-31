# 实现硬盘用量查询 API

在 registerLicenseHandler 里添加新的路由处理函数：

```go
s.mux.HandleFunc("/api/license/disks", s.wrapContentHandler(s.handler.GetDisksHandler))
```

## 用户会传入 key

licenseKey := req.URL.Query().Get("key")

## 根据 key 查询 License 信息

// 验证 License 是否存在
license, err := s.contentApp.GetLicenseByKey(licenseKey)

## 获取 SyncAccount 里的 CouchDB 信息

syncAccount, _ := s.contentApp.GetSyncAccountByLicense(license.LicenseKey)

为 s.couchClient 添加新方法 GetDisks， 获取硬盘用量信息。


## 获取 publish 用到的文件目录的硬盘用量

先获取用户 的 publish 目录路径： filepath.Join(application.PreviewDir(), s.db.UserDir())
其中 application.PreviewDir() 是发布目录的根路径， s.db.UserDir() 是用户目录。

得到路径后，使用系统命令查看硬盘用量， 我们的开发环境是 MacOS, 我们的生产环境是 Ubuntu Linux, 这两个系统都支持 df 命令。

参考下面的代码：

```go
func DirSizeByDU(path string) (int64, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("du", "-sb", path)
	case "darwin":
		cmd = exec.Command("du", "-sk", path)
	default:
		return 0, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return 0, fmt.Errorf("unexpected du output: %s", out)
	}

	size, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0, err
	}

	if runtime.GOOS == "darwin" {
		size *= 1024
	}

	return size, nil
}
```


最后返回的 JSON 格式里，以 M 为单位， 返回 couchdb_disk_usage 和 publish_disk_usage 两个字段， 以及总的字段 total_disk_usage。
