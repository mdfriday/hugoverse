package application

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mdfriday/hugoverse/internal/infrastructure/caddy"
	"github.com/mdfriday/hugoverse/internal/infrastructure/couchdb"
	"github.com/mdfriday/hugoverse/pkg/loggers"
	bolt "go.etcd.io/bbolt"
)

// BackupScheduler 备份调度器
type BackupScheduler struct {
	backupDir     string              // ~/data_bk
	dataDir       string              // ~/.local/share/hugoverse
	retention     int                 // 保留天数（默认 7 天）
	maxRetries    int                 // 最大重试次数（默认 3 次）
	retryDelay    time.Duration       // 重试间隔（默认 1 分钟）
	
	couchdbClient *couchdb.Client
	caddyClient   *caddy.Client
	systemDBPath  string
	
	log           loggers.Logger
}

// BackupMetadata 备份元数据
type BackupMetadata struct {
	Version       string              `json:"version"`
	BackupType    string              `json:"backup_type"`
	Timestamp     time.Time           `json:"timestamp"`
	Hostname      string              `json:"hostname"`
	Service       string              `json:"service"`
	CouchDB       CouchDBBackupInfo   `json:"couchdb"`
	Caddy         CaddyBackupInfo     `json:"caddy"`
	Hugoverse     HugoverseBackupInfo `json:"hugoverse"`
	Compression   CompressionInfo     `json:"compression"`
	TotalDuration int64               `json:"total_duration_ms"`
	Status        string              `json:"status"`
}

// CouchDBBackupInfo CouchDB 备份信息
type CouchDBBackupInfo struct {
	Status        string                       `json:"status"`
	URL           string                       `json:"url"`
	Databases     map[string]DatabaseBackupInfo `json:"databases"`
	TotalDatabases int                         `json:"total_databases"`
	TotalDocs     int                          `json:"total_docs"`
	TotalSize     int64                        `json:"total_size_bytes"`
	Duration      int64                        `json:"total_duration_ms"`
}

// DatabaseBackupInfo 单个数据库备份信息
type DatabaseBackupInfo struct {
	DocCount   int    `json:"doc_count"`
	UpdateSeq  string `json:"update_seq"`
	Size       int64  `json:"size_bytes"`
	BackupTime int64  `json:"backup_time_ms"`
}

// CaddyBackupInfo Caddy 备份信息
type CaddyBackupInfo struct {
	Status   string `json:"status"`
	Size     int64  `json:"size_bytes"`
	Duration int64  `json:"duration_ms"`
}

// HugoverseBackupInfo Hugoverse 备份信息
type HugoverseBackupInfo struct {
	Status   string `json:"status"`
	DBPath   string `json:"db_path"`
	Size     int64  `json:"size_bytes"`
	Duration int64  `json:"duration_ms"`
}

// CompressionInfo 压缩信息
type CompressionInfo struct {
	OriginalSize   int64   `json:"original_size_bytes"`
	CompressedSize int64   `json:"compressed_size_bytes"`
	Ratio          float64 `json:"ratio"`
	Algorithm      string  `json:"algorithm"`
}

// CaddyStartParams Caddy 启动参数
type CaddyStartParams struct {
	Domain       string `json:"domain"`
	DNSPodToken  string `json:"dnspod_token"`
	ServerIP     string `json:"server_ip"`
	Backend      string `json:"backend"`
	CouchDB      string `json:"couchdb"`
	PIDFile      string `json:"pid_file"`
	LogFile      string `json:"log_file"`
}

// NewBackupScheduler 创建备份调度器
func NewBackupScheduler(couchdbCfg *couchdb.Config, caddyCfg *caddy.Config, log loggers.Logger) *BackupScheduler {
	homeDir, _ := os.UserHomeDir()
	backupDir := filepath.Join(homeDir, "data_bk")
	
	// 确保备份目录存在
	os.MkdirAll(backupDir, 0700) // 只有 owner 可以访问
	
	dataDir := DataDir()
	systemDBPath := filepath.Join(dataDir, "system.db")
	
	return &BackupScheduler{
		backupDir:     backupDir,
		dataDir:       dataDir,
		retention:     1,  // 只保留 1 天（磁盘空间有限）
		maxRetries:    3,
		retryDelay:    1 * time.Minute,
		couchdbClient: couchdb.NewClient(couchdbCfg),
		caddyClient:   caddy.NewClient(caddyCfg),
		systemDBPath:  systemDBPath,
		log:           log,
	}
}

// Start 启动备份定时任务
func (bs *BackupScheduler) Start() {
	// 计算到凌晨 2:00（中国时间）的时间
	loc := time.FixedZone("CST", 8*3600)
	now := time.Now().In(loc)
	
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, loc)
	if now.After(nextRun) {
		nextRun = nextRun.Add(24 * time.Hour)
	}
	
	bs.log.Printf("Backup scheduler started, next run at: %s", nextRun.Format(time.RFC3339))
	
	// 等待到首次执行时间
	time.Sleep(time.Until(nextRun))
	
	// 创建 24 小时定时器
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	
	// 首次执行
	bs.executeBackup()
	
	// 定时执行
	for range ticker.C {
		bs.executeBackup()
	}
}

// executeBackup 执行完整备份流程
func (bs *BackupScheduler) executeBackup() {
	startTime := time.Now()
	backupID := fmt.Sprintf("backup-%s", startTime.Format("2006-01-02"))
	
	bs.log.Printf("====== Backup Started ======")
	bs.log.Printf("Backup ID: %s", backupID)
	
	// 检查磁盘空间（包括 CouchDB 数据大小估算）
	if err := bs.checkDiskSpace(); err != nil {
		bs.log.Errorf("Disk space check failed: %v", err)
		bs.log.Errorf("Backup aborted due to insufficient disk space")
		return
	}
	
	// 创建临时目录
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("hugoverse-backup-%s", startTime.Format("20060102150405")))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		bs.log.Errorf("Failed to create temp directory: %v", err)
		return
	}
	defer os.RemoveAll(tempDir)
	
	bs.log.Printf("Creating temp directory: %s", tempDir)
	
	// 初始化元数据
	metadata := &BackupMetadata{
		Version:    "1.0",
		BackupType: "full",
		Timestamp:  startTime,
		Hostname:   bs.getHostname(),
		Service:    "obsidian-sync",
		Status:     "in_progress",
	}
	
	// Phase 1: 备份 CouchDB
	bs.log.Printf("")
	bs.log.Printf("====== Phase 1: CouchDB Backup ======")
	couchdbStart := time.Now()
	if err := bs.backupWithRetry(func() error {
		return bs.backupCouchDB(tempDir, &metadata.CouchDB)
	}); err != nil {
		bs.log.Errorf("CouchDB backup failed after retries: %v", err)
		metadata.CouchDB.Status = "failed"
	} else {
		metadata.CouchDB.Status = "success"
		metadata.CouchDB.Duration = time.Since(couchdbStart).Milliseconds()
	}
	
	// Phase 2: 备份 Caddy
	bs.log.Printf("")
	bs.log.Printf("====== Phase 2: Caddy Backup ======")
	caddyStart := time.Now()
	if err := bs.backupWithRetry(func() error {
		return bs.backupCaddy(tempDir, &metadata.Caddy)
	}); err != nil {
		bs.log.Errorf("Caddy backup failed after retries: %v", err)
		metadata.Caddy.Status = "failed"
	} else {
		metadata.Caddy.Status = "success"
		metadata.Caddy.Duration = time.Since(caddyStart).Milliseconds()
	}
	
	// Phase 3: 备份 Hugoverse
	bs.log.Printf("")
	bs.log.Printf("====== Phase 3: Hugoverse Backup ======")
	hugoverseStart := time.Now()
	if err := bs.backupWithRetry(func() error {
		return bs.backupSystemDB(tempDir, &metadata.Hugoverse)
	}); err != nil {
		bs.log.Errorf("Hugoverse backup failed after retries: %v", err)
		metadata.Hugoverse.Status = "failed"
	} else {
		metadata.Hugoverse.Status = "success"
		metadata.Hugoverse.Duration = time.Since(hugoverseStart).Milliseconds()
	}
	
	// Phase 4: 压缩打包
	bs.log.Printf("")
	bs.log.Printf("====== Phase 4: Compression ======")
	
	// 生成文档
	if err := bs.generateMetadata(tempDir, metadata); err != nil {
		bs.log.Errorf("Failed to generate metadata: %v", err)
	}
	
	if err := bs.generateRestoreScript(tempDir); err != nil {
		bs.log.Errorf("Failed to generate restore script: %v", err)
	}
	
	if err := bs.generateRestoreDoc(tempDir, metadata); err != nil {
		bs.log.Errorf("Failed to generate restore doc: %v", err)
	}
	
	// 创建归档
	archivePath := filepath.Join(bs.backupDir, backupID+".tar.gz")
	if err := bs.createArchive(tempDir, archivePath, &metadata.Compression); err != nil {
		bs.log.Errorf("Failed to create archive: %v", err)
		metadata.Status = "failed"
		return
	}
	
	// Phase 5: 验证
	bs.log.Printf("")
	bs.log.Printf("====== Phase 5: Verification ======")
	if err := bs.verifyBackup(archivePath); err != nil {
		bs.log.Errorf("Backup verification failed: %v", err)
		metadata.Status = "failed"
		return
	}
	
	// Phase 6: 清理
	bs.log.Printf("")
	bs.log.Printf("====== Phase 6: Cleanup ======")
	if err := bs.updateLatestSymlink(archivePath); err != nil {
		bs.log.Warnf("Failed to update 'latest' symlink: %v", err)
	}
	
	if err := bs.cleanupOldBackups(); err != nil {
		bs.log.Warnf("Failed to cleanup old backups: %v", err)
	}
	
	// 更新元数据
	metadata.TotalDuration = time.Since(startTime).Milliseconds()
	metadata.Status = "success"
	
	// 输出摘要
	bs.log.Printf("")
	bs.log.Printf("====== Backup Summary ======")
	bs.log.Printf("Status: %s", strings.ToUpper(metadata.Status))
	bs.log.Printf("Duration: %d seconds", metadata.TotalDuration/1000)
	bs.log.Printf("Components:")
	bs.log.Printf("  - CouchDB: %d databases, %d docs, %.2f MB",
		metadata.CouchDB.TotalDatabases,
		metadata.CouchDB.TotalDocs,
		float64(metadata.CouchDB.TotalSize)/1024/1024)
	bs.log.Printf("  - Caddy: %.2f KB", float64(metadata.Caddy.Size)/1024)
	bs.log.Printf("  - Hugoverse: %.2f MB", float64(metadata.Hugoverse.Size)/1024/1024)
	bs.log.Printf("Total size: %.2f MB (original) -> %.2f MB (compressed)",
		float64(metadata.Compression.OriginalSize)/1024/1024,
		float64(metadata.Compression.CompressedSize)/1024/1024)
	bs.log.Printf("Backup location: %s", archivePath)
	bs.log.Printf("====== Backup Completed ======")
}

// backupWithRetry 带重试的备份执行
func (bs *BackupScheduler) backupWithRetry(backupFunc func() error) error {
	var lastErr error
	
	for attempt := 1; attempt <= bs.maxRetries; attempt++ {
		err := backupFunc()
		if err == nil {
			return nil
		}
		
		lastErr = err
		bs.log.Errorf("Backup attempt %d/%d failed: %v", attempt, bs.maxRetries, err)
		
		if attempt < bs.maxRetries {
			bs.log.Printf("Retrying in %v...", bs.retryDelay)
			time.Sleep(bs.retryDelay)
		}
	}
	
	return fmt.Errorf("backup failed after %d attempts: %w", bs.maxRetries, lastErr)
}

// backupCouchDB 备份 CouchDB 数据库
func (bs *BackupScheduler) backupCouchDB(tempDir string, info *CouchDBBackupInfo) error {
	bs.log.Printf("Connecting to CouchDB at %s", bs.couchdbClient.GetURL())
	
	couchdbDir := filepath.Join(tempDir, "couchdb")
	if err := os.MkdirAll(couchdbDir, 0755); err != nil {
		return fmt.Errorf("failed to create couchdb directory: %w", err)
	}
	
	// 获取所有数据库列表
	databases, err := bs.listCouchDBDatabases()
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}
	
	// 过滤需要备份的数据库：_users 和 userdb-*
	var targetDatabases []string
	for _, db := range databases {
		if db == "_users" || strings.HasPrefix(db, "userdb-") {
			targetDatabases = append(targetDatabases, db)
		}
	}
	
	bs.log.Printf("Found %d databases (%d target databases for backup)", len(databases), len(targetDatabases))
	
	// 初始化信息
	info.URL = bs.couchdbClient.GetURL()
	info.Databases = make(map[string]DatabaseBackupInfo)
	
	// 并行备份数据库
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)
	
	for _, dbName := range targetDatabases {
		wg.Add(1)
		go func(db string) {
			defer wg.Done()
			
			dbInfo, err := bs.backupCouchDBDatabase(couchdbDir, db)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("database %s: %w", db, err))
				mu.Unlock()
				return
			}
			
			mu.Lock()
			info.Databases[db] = dbInfo
			info.TotalDatabases++
			info.TotalDocs += dbInfo.DocCount
			info.TotalSize += dbInfo.Size
			mu.Unlock()
		}(dbName)
	}
	
	wg.Wait()
	
	if len(errors) > 0 {
		return fmt.Errorf("failed to backup %d databases: %v", len(errors), errors[0])
	}
	
	bs.log.Printf("CouchDB backup completed: %d databases, %d docs, %.2f MB",
		info.TotalDatabases, info.TotalDocs, float64(info.TotalSize)/1024/1024)
	
	return nil
}

// backupCouchDBDatabase 备份单个 CouchDB 数据库
func (bs *BackupScheduler) backupCouchDBDatabase(dir string, dbName string) (DatabaseBackupInfo, error) {
	start := time.Now()
	
	// 获取数据库信息
	dbInfo, err := bs.getCouchDBDatabaseInfo(dbName)
	if err != nil {
		return DatabaseBackupInfo{}, err
	}
	
	bs.log.Printf("Backing up: %s (%d docs)", dbName, dbInfo["doc_count"])
	
	// 导出所有文档
	docs, err := bs.exportCouchDBDocs(dbName)
	if err != nil {
		return DatabaseBackupInfo{}, err
	}
	
	// 构建备份数据
	backup := map[string]interface{}{
		"db_name":    dbName,
		"update_seq": dbInfo["update_seq"],
		"doc_count":  dbInfo["doc_count"],
		"timestamp":  time.Now().Format(time.RFC3339),
		"docs":       docs,
	}
	
	// 保存为 JSON 文件
	outputPath := filepath.Join(dir, dbName+".json")
	data, err := json.Marshal(backup)
	if err != nil {
		return DatabaseBackupInfo{}, fmt.Errorf("failed to marshal backup: %w", err)
	}
	
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return DatabaseBackupInfo{}, fmt.Errorf("failed to write backup file: %w", err)
	}
	
	info := DatabaseBackupInfo{
		DocCount:   int(dbInfo["doc_count"].(float64)),
		UpdateSeq:  fmt.Sprintf("%v", dbInfo["update_seq"]),
		Size:       int64(len(data)),
		BackupTime: time.Since(start).Milliseconds(),
	}
	
	bs.log.Printf("✓ %s backed up: %.2f KB", dbName, float64(info.Size)/1024)
	
	return info, nil
}

// backupCaddy 备份 Caddy 配置
func (bs *BackupScheduler) backupCaddy(tempDir string, info *CaddyBackupInfo) error {
	bs.log.Printf("Exporting Caddy configuration from Admin API...")
	
	caddyDir := filepath.Join(tempDir, "caddy")
	if err := os.MkdirAll(caddyDir, 0755); err != nil {
		return fmt.Errorf("failed to create caddy directory: %w", err)
	}
	
	// 导出配置
	configPath := filepath.Join(caddyDir, "caddy-config.json")
	if err := bs.caddyClient.ExportConfig(configPath); err != nil {
		return fmt.Errorf("failed to export caddy config: %w", err)
	}
	
	stat, _ := os.Stat(configPath)
	info.Size = stat.Size()
	
	bs.log.Printf("✓ Caddy config backed up: %.2f KB", float64(info.Size)/1024)
	
	// 保存启动参数
	bs.log.Printf("Saving Caddy start parameters...")
	params := CaddyStartParams{
		Domain:      bs.caddyClient.GetConfigObject().CoreDomain,
		DNSPodToken: bs.caddyClient.GetConfigObject().DNSPodToken,
		ServerIP:    bs.caddyClient.GetConfigObject().ServerIP,
		Backend:     bs.caddyClient.GetConfigObject().DefaultBackend,
		CouchDB:     bs.caddyClient.GetConfigObject().CouchDBBackend,
		PIDFile:     bs.caddyClient.GetConfigObject().PidFile,
		LogFile:     bs.caddyClient.GetConfigObject().LogFile,
	}
	
	paramsData, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal start params: %w", err)
	}
	
	paramsPath := filepath.Join(caddyDir, "start-params.json")
	if err := os.WriteFile(paramsPath, paramsData, 0644); err != nil {
		return fmt.Errorf("failed to write start params: %w", err)
	}
	
	bs.log.Printf("✓ Start parameters saved")
	
	return nil
}

// backupSystemDB 备份 Hugoverse system.db
func (bs *BackupScheduler) backupSystemDB(tempDir string, info *HugoverseBackupInfo) error {
	bs.log.Printf("Backing up system.db...")
	
	hugoverseDir := filepath.Join(tempDir, "hugoverse")
	if err := os.MkdirAll(hugoverseDir, 0755); err != nil {
		return fmt.Errorf("failed to create hugoverse directory: %w", err)
	}
	
	// 打开数据库（只读）
	db, err := bolt.Open(bs.systemDBPath, 0666, &bolt.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open system.db: %w", err)
	}
	defer db.Close()
	
	// 备份文件路径
	backupPath := filepath.Join(hugoverseDir, "system.db")
	backupFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer backupFile.Close()
	
	// 使用 BoltDB 的内置备份功能
	err = db.View(func(tx *bolt.Tx) error {
		_, err := tx.WriteTo(backupFile)
		return err
	})
	
	if err != nil {
		return fmt.Errorf("failed to backup database: %w", err)
	}
	
	stat, _ := os.Stat(backupPath)
	info.DBPath = bs.systemDBPath
	info.Size = stat.Size()
	
	bs.log.Printf("✓ system.db backed up: %.2f MB", float64(info.Size)/1024/1024)
	
	return nil
}

// createArchive 创建压缩归档
func (bs *BackupScheduler) createArchive(sourceDir, archivePath string, compression *CompressionInfo) error {
	bs.log.Printf("Creating archive: %s", filepath.Base(archivePath))
	
	// 计算原始大小
	originalSize, err := bs.getDirSize(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to calculate directory size: %w", err)
	}
	
	// 创建归档文件
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer archiveFile.Close()
	
	// 创建 gzip writer
	gzipWriter := gzip.NewWriter(archiveFile)
	defer gzipWriter.Close()
	
	// 创建 tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()
	
	// 遍历源目录，添加文件到归档
	baseDir := filepath.Base(sourceDir)
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// 创建 tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		
		// 设置相对路径
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		header.Name = filepath.Join(baseDir, relPath)
		
		// 写入 header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		
		// 如果是文件，写入内容
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			
			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}
		}
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to create tar archive: %w", err)
	}
	
	// 关闭 writers 以确保所有数据被写入
	tarWriter.Close()
	gzipWriter.Close()
	archiveFile.Close()
	
	// 获取压缩后的大小
	stat, _ := os.Stat(archivePath)
	compressedSize := stat.Size()
	
	compression.OriginalSize = originalSize
	compression.CompressedSize = compressedSize
	compression.Ratio = float64(compressedSize) / float64(originalSize)
	compression.Algorithm = "gzip"
	
	bs.log.Printf("Compression completed: %.2f MB -> %.2f MB (%.0f%%)",
		float64(originalSize)/1024/1024,
		float64(compressedSize)/1024/1024,
		compression.Ratio*100)
	
	return nil
}

// 辅助方法继续在下一部分...
