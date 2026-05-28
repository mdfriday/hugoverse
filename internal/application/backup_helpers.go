package application

// 此文件包含 backup.go 的辅助方法

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
)

// 辅助方法

// checkDiskSpace 检查磁盘空间
func (bs *BackupScheduler) checkDiskSpace() error {
	bs.log.Printf("Checking disk space and estimating backup size...")

	// 1. 获取当前可用磁盘空间
	var stat syscall.Statfs_t
	if err := syscall.Statfs(bs.backupDir, &stat); err != nil {
		return fmt.Errorf("failed to get filesystem stats: %w", err)
	}

	availableSpace := stat.Bavail * uint64(stat.Bsize)
	availableGB := float64(availableSpace) / 1024 / 1024 / 1024

	bs.log.Printf("Available disk space: %.2f GB", availableGB)

	// 2. 估算 CouchDB 数据大小
	couchdbSize, err := bs.estimateCouchDBSize()
	if err != nil {
		bs.log.Warnf("Failed to estimate CouchDB size: %v, using default estimate", err)
		couchdbSize = 1024 * 1024 * 1024 // 默认估算 1GB
	}

	couchdbSizeGB := float64(couchdbSize) / 1024 / 1024 / 1024
	bs.log.Printf("Estimated CouchDB data size: %.2f GB", couchdbSizeGB)

	// 3. 估算 system.db 大小
	systemDBSize := int64(0)
	if stat, err := os.Stat(bs.systemDBPath); err == nil {
		systemDBSize = stat.Size()
	}
	systemDBSizeMB := float64(systemDBSize) / 1024 / 1024
	bs.log.Printf("Hugoverse system.db size: %.2f MB", systemDBSizeMB)

	// 4. 计算预期备份大小（原始数据 + 10% 余量，压缩后约 20-30%）
	totalDataSize := couchdbSize + systemDBSize
	// 压缩率约 20-30%，加上临时文件，需要约 1.5 倍空间
	estimatedBackupSize := int64(float64(totalDataSize) * 1.5)
	estimatedBackupGB := float64(estimatedBackupSize) / 1024 / 1024 / 1024

	bs.log.Printf("Estimated backup size (with compression and temp files): %.2f GB", estimatedBackupGB)

	// 5. 检查是否有足够空间
	// 考虑到只保留 1 天备份，需要：新备份 + 旧备份（如果存在）+ 1GB 余量
	// 保守估计：需要 2 倍备份大小 + 1GB
	requiredSpace := estimatedBackupGB*2 + 1.0

	if availableGB < requiredSpace {
		return fmt.Errorf(
			"insufficient disk space: %.2f GB available, need at least %.2f GB (%.2f GB × 2 for new+old backup + 1 GB buffer). Current disk usage: ~30GB used, ~30GB available",
			availableGB,
			requiredSpace,
			estimatedBackupGB,
		)
	}

	bs.log.Printf("Disk space check passed: %.2f GB available, %.2f GB required (for 2 backups + buffer)", availableGB, requiredSpace)

	return nil
}

// estimateCouchDBSize 估算 CouchDB 数据大小
func (bs *BackupScheduler) estimateCouchDBSize() (int64, error) {
	// 获取所有数据库列表
	databases, err := bs.listCouchDBDatabases()
	if err != nil {
		return 0, fmt.Errorf("failed to list databases: %w", err)
	}

	// 获取付费用户的数据库列表
	paidUserDBs := bs.getPaidUserDatabases()

	// 过滤需要备份的数据库：_users 和付费用户的 userdb-*
	var targetDatabases []string
	for _, db := range databases {
		if db == "_users" {
			targetDatabases = append(targetDatabases, db)
		} else if strings.HasPrefix(db, "userdb-") {
			if _, isPaid := paidUserDBs[db]; isPaid {
				targetDatabases = append(targetDatabases, db)
			}
		}
	}

	bs.log.Printf("Calculating size for %d databases (%d paid users + _users)...", len(targetDatabases), len(targetDatabases)-1)

	// 计算总大小
	var totalSize int64
	for _, dbName := range targetDatabases {
		size, err := bs.couchdbClient.GetDiskUsage(dbName)
		if err != nil {
			bs.log.Warnf("Failed to get size for database %s: %v", dbName, err)
			continue
		}
		totalSize += size
	}

	return totalSize, nil
}

// getHostname 获取主机名
func (bs *BackupScheduler) getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// getDirSize 计算目录大小
func (bs *BackupScheduler) getDirSize(dir string) (int64, error) {
	var size int64

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// verifyBackup 验证备份完整性
func (bs *BackupScheduler) verifyBackup(archivePath string) error {
	bs.log.Printf("Verifying archive integrity...")

	// 检查文件是否存在
	stat, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("archive file not found: %w", err)
	}

	// 检查文件大小
	if stat.Size() == 0 {
		return fmt.Errorf("archive file is empty")
	}

	// 尝试列出归档内容
	cmd := exec.Command("tar", "-tzf", archivePath)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to verify archive: %w", err)
	}

	// 检查必要的文件是否存在
	files := string(output)
	requiredFiles := []string{"metadata.json", "RESTORE.md", "restore.sh"}
	for _, required := range requiredFiles {
		if !strings.Contains(files, required) {
			return fmt.Errorf("required file missing in archive: %s", required)
		}
	}

	bs.log.Printf("✓ Archive is valid")
	bs.log.Printf("✓ All expected files present")

	return nil
}

// updateLatestSymlink 更新 latest 符号链接
func (bs *BackupScheduler) updateLatestSymlink(archivePath string) error {
	bs.log.Printf("Updating 'latest' symlink...")

	latestLink := filepath.Join(bs.backupDir, "latest")

	// 删除旧的符号链接
	os.Remove(latestLink)

	// 创建新的符号链接
	if err := os.Symlink(filepath.Base(archivePath), latestLink); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// cleanupOldBackups 清理旧备份
func (bs *BackupScheduler) cleanupOldBackups() error {
	bs.log.Printf("Cleaning old backups (keeping %d day)...", bs.retention)

	files, err := filepath.Glob(filepath.Join(bs.backupDir, "backup-*.tar.gz"))
	if err != nil {
		return fmt.Errorf("failed to list backup files: %w", err)
	}

	if len(files) <= bs.retention {
		bs.log.Printf("No old backups to delete (total: %d files)", len(files))
		return nil
	}

	cutoff := time.Now().AddDate(0, 0, -bs.retention)
	deleted := 0

	for _, file := range files {
		stat, err := os.Stat(file)
		if err != nil {
			continue
		}

		if stat.ModTime().Before(cutoff) {
			age := int(time.Since(stat.ModTime()).Hours() / 24)
			bs.log.Printf("Deleting old backup: %s (%d days old, %.2f GB)",
				filepath.Base(file), age, float64(stat.Size())/1024/1024/1024)

			if err := os.Remove(file); err != nil {
				bs.log.Warnf("Failed to delete %s: %v", file, err)
			} else {
				deleted++
			}
		}
	}

	if deleted > 0 {
		bs.log.Printf("Deleted %d old backup(s), freed approximately %.2f GB", deleted, 0.0)
	}

	return nil
}

// generateMetadata 生成元数据文件
func (bs *BackupScheduler) generateMetadata(dir string, metadata *BackupMetadata) error {
	bs.log.Printf("Generating metadata.json...")

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metadataPath := filepath.Join(dir, "metadata.json")
	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// generateRestoreDoc 生成恢复文档
func (bs *BackupScheduler) generateRestoreDoc(dir string, metadata *BackupMetadata) error {
	bs.log.Printf("Generating RESTORE.md...")

	doc := fmt.Sprintf(`# Obsidian 同步服务 - 数据恢复指南

## ⚠️ 重要提醒

本备份包含 **Obsidian 文件同步服务** 的数据。恢复前请仔细阅读本文档。

---

## 📊 备份信息

- **备份时间**：%s
- **服务器**：%s
- **用户数量**：%d 个
- **总文档数**：%d 个文档
- **原始大小**：%.2f MB
- **压缩大小**：%.2f MB（压缩率 %.0f%%）
- **备份耗时**：%d 秒

### 数据库详情

| 数据库 | 文档数 | 大小 |
|--------|--------|------|
`,
		metadata.Timestamp.Format("2006-01-02 15:04:05 MST"),
		metadata.Hostname,
		metadata.CouchDB.TotalDatabases,
		metadata.CouchDB.TotalDocs,
		float64(metadata.Compression.OriginalSize)/1024/1024,
		float64(metadata.Compression.CompressedSize)/1024/1024,
		metadata.Compression.Ratio*100,
		metadata.TotalDuration/1000,
	)

	// 添加数据库详情
	for dbName, dbInfo := range metadata.CouchDB.Databases {
		doc += fmt.Sprintf("| %s | %d | %.2f MB |\n",
			dbName, dbInfo.DocCount, float64(dbInfo.Size)/1024/1024)
	}

	doc += `

---

## 🚀 快速恢复（完全灾难恢复）

**适用场景**：服务器重建，CouchDB 是全新的，没有用户数据

` + "```bash" + `
# 1. 解压备份
cd ~/data_bk
tar -xzf backup-YYYY-MM-DD.tar.gz
cd backup-YYYY-MM-DD

# 2. 停止相关服务
pkill -f hugoverse
go/bin/hugoverse caddy stop

# 3. 执行完整恢复
chmod +x restore.sh
./restore.sh all

# 4. 启动服务
# Caddy
go/bin/hugoverse caddy start \
  -domain mdfriday.com \
  -dnspod-token YOUR_TOKEN \
  -server-ip YOUR_IP

# Hugoverse
nohup go/bin/hugoverse serve -env prod &

# 5. 验证
curl http://localhost:5984/_all_dbs
ps aux | grep hugoverse
` + "```" + `

---

## 🔧 高级恢复选项

### 1. 只恢复某个用户的数据

` + "```bash" + `
./restore.sh couchdb --database userdb-user@example.com
` + "```" + `

### 2. 只恢复 Caddy 配置

` + "```bash" + `
./restore.sh caddy
` + "```" + `

### 3. 只恢复 Hugoverse 数据库

` + "```bash" + `
./restore.sh hugoverse
` + "```" + `

---

## ✅ 验证恢复结果

` + "```bash" + `
# 检查 CouchDB
curl http://localhost:5984/_all_dbs

# 检查 Caddy
curl http://127.0.0.1:2019/config/

# 检查 Hugoverse
ps aux | grep hugoverse
` + "```" + `

---

## 📞 技术支持

如有问题，请查看：
- 备份日志：` + "`cat ~/data_bk/backup.log`" + `
- 恢复日志：` + "`cat restore.log`" + `
- 完整文档：prompts/backup-and-restore-solution.md
`

	docPath := filepath.Join(dir, "RESTORE.md")
	if err := os.WriteFile(docPath, []byte(doc), 0644); err != nil {
		return fmt.Errorf("failed to write restore doc: %w", err)
	}

	return nil
}

// generateRestoreScript 生成恢复脚本
func (bs *BackupScheduler) generateRestoreScript(dir string) error {
	bs.log.Printf("Generating restore.sh...")

	script := `#!/bin/bash
# Auto-generated restore script for Hugoverse backup
# Generated at: ` + time.Now().Format(time.RFC3339) + `

set -e

BACKUP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPONENT="${1:-all}"
COUCHDB_URL="${COUCHDB_URL:-http://localhost:5984}"
COUCHDB_ADMIN="${COUCHDB_ADMIN:-admin}"
COUCHDB_PASSWORD="${COUCHDB_PASSWORD:-password}"

log_info() {
    echo "[INFO] $1"
}

log_success() {
    echo "[SUCCESS] $1"
}

log_error() {
    echo "[ERROR] $1" >&2
}

# ====================================
# CouchDB 恢复
# ====================================
restore_couchdb() {
    log_info "Restoring CouchDB databases..."
    
    cd "$BACKUP_DIR/couchdb"
    
    local total_dbs=0
    local total_docs=0
    
    for file in *.json; do
        [ -f "$file" ] || continue
        
        db=$(basename "$file" .json)
        log_info "  - Creating database: $db"
        
        # 创建数据库（忽略已存在错误）
        curl -s -u "$COUCHDB_ADMIN:$COUCHDB_PASSWORD" \
            -X PUT "$COUCHDB_URL/$db" > /dev/null 2>&1 || true
        
        # 解析 JSON，提取 docs 数组
        docs=$(jq -c '.docs' "$file")
        doc_count=$(echo "$docs" | jq 'length')
        
        log_info "  - Importing $doc_count docs to $db..."
        
        # 批量导入文档
        echo "{\"docs\": $docs}" | curl -s -u "$COUCHDB_ADMIN:$COUCHDB_PASSWORD" \
            -X POST "$COUCHDB_URL/$db/_bulk_docs" \
            -H "Content-Type: application/json" \
            -d @- > /dev/null
        
        if [ $? -eq 0 ]; then
            log_info "  - ✓ $db restored successfully"
            total_dbs=$((total_dbs + 1))
            total_docs=$((total_docs + doc_count))
        else
            log_error "  - ✗ Failed to restore $db"
        fi
    done
    
    log_success "CouchDB restore completed ($total_dbs databases, $total_docs docs)"
}

# ====================================
# Caddy 恢复
# ====================================
restore_caddy() {
    log_info "Restoring Caddy configuration..."
    
    cd "$BACKUP_DIR/caddy"
    
    # 读取启动参数
    if [ -f "start-params.json" ]; then
        DOMAIN=$(jq -r '.domain' start-params.json)
        DNSPOD_TOKEN=$(jq -r '.dnspod_token' start-params.json)
        # 旧备份可能没有 dns_provider，默认按 tencentcloud 兼容
        DNS_PROVIDER=$(jq -r '.dns_provider // "tencentcloud"' start-params.json)
        SERVER_IP=$(jq -r '.server_ip' start-params.json)

        log_info "  Domain: $DOMAIN"
        log_info "  Server IP: $SERVER_IP"
        log_info "  DNS Provider: $DNS_PROVIDER"
        log_info "  DNS Token: ***configured***"

        # 停止旧的 Caddy
        pkill -f caddy || true
        sleep 2

        # 使用备份的参数启动
        log_info "  Starting Caddy..."
        go/bin/hugoverse caddy start \
            -domain "$DOMAIN" \
            -dns-provider "$DNS_PROVIDER" \
            -dnspod-token "$DNSPOD_TOKEN" \
            -server-ip "$SERVER_IP" > /dev/null 2>&1 &
        
        sleep 3
        
        # 验证启动
        if curl -s http://127.0.0.1:2019/config/ > /dev/null; then
            log_success "Caddy restored and started successfully"
        else
            log_error "Failed to start Caddy"
            return 1
        fi
    else
        log_error "start-params.json not found"
        return 1
    fi
}

# ====================================
# Hugoverse 恢复
# ====================================
restore_hugoverse() {
    log_info "Restoring Hugoverse database..."
    
    cd "$BACKUP_DIR/hugoverse"
    
    # 确保目标目录存在
    HUGOVERSE_DIR="${HUGOVERSE_DATA_DIR:-$HOME/.local/share/hugoverse}"
    mkdir -p "$HUGOVERSE_DIR"
    
    # 备份现有数据库
    if [ -f "$HUGOVERSE_DIR/system.db" ]; then
        log_info "  - Backing up existing system.db..."
        cp "$HUGOVERSE_DIR/system.db" "$HUGOVERSE_DIR/system.db.bak.$(date +%Y%m%d_%H%M%S)"
    fi
    
    # 复制数据库
    log_info "  - Copying system.db..."
    cp system.db "$HUGOVERSE_DIR/system.db"
    
    if [ $? -eq 0 ]; then
        log_success "Hugoverse database restored successfully"
    else
        log_error "Failed to restore Hugoverse database"
        return 1
    fi
}

# ====================================
# 主恢复流程
# ====================================
case "$COMPONENT" in
    couchdb)
        restore_couchdb
        ;;
    caddy)
        restore_caddy
        ;;
    hugoverse)
        restore_hugoverse
        ;;
    all)
        restore_couchdb
        restore_caddy
        restore_hugoverse
        log_success "Full restore completed successfully!"
        echo ""
        echo "Next steps:"
        echo "1. Verify CouchDB: curl http://localhost:5984/_all_dbs"
        echo "2. Verify Caddy: curl http://127.0.0.1:2019/config/"
        echo "3. Start Hugoverse: nohup go/bin/hugoverse serve -env prod &"
        echo "4. Test Obsidian sync from client"
        ;;
    *)
        echo "Usage: $0 {couchdb|caddy|hugoverse|all}"
        exit 1
        ;;
esac
`

	scriptPath := filepath.Join(dir, "restore.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("failed to write restore script: %w", err)
	}

	return nil
}

// CouchDB 辅助方法

// listCouchDBDatabases 列出所有数据库
func (bs *BackupScheduler) listCouchDBDatabases() ([]string, error) {
	url := fmt.Sprintf("%s/_all_dbs", bs.couchdbClient.GetURL())

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	bs.setBasicAuth(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var databases []string
	if err := json.NewDecoder(resp.Body).Decode(&databases); err != nil {
		return nil, err
	}

	return databases, nil
}

// getCouchDBDatabaseInfo 获取数据库信息
func (bs *BackupScheduler) getCouchDBDatabaseInfo(dbName string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/%s", bs.couchdbClient.GetURL(), dbName)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	bs.setBasicAuth(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var info map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	return info, nil
}

// exportCouchDBDocs 导出数据库所有文档
func (bs *BackupScheduler) exportCouchDBDocs(dbName string) ([]interface{}, error) {
	url := fmt.Sprintf("%s/%s/_all_docs?include_docs=true", bs.couchdbClient.GetURL(), dbName)
	
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	
	bs.setBasicAuth(req)
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}
	
	var result struct {
		Rows []struct {
			Doc interface{} `json:"doc"`
		} `json:"rows"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	docs := make([]interface{}, 0, len(result.Rows))
	for _, row := range result.Rows {
		if row.Doc != nil {
			docs = append(docs, row.Doc)
		}
	}
	
	return docs, nil
}

// streamExportCouchDBDocs 流式导出数据库文档（分批处理，避免 OOM）
func (bs *BackupScheduler) streamExportCouchDBDocs(dbName string, dbInfo map[string]interface{}, outputPath string) (int64, error) {
	// 创建输出文件
	file, err := os.Create(outputPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	
	// 写入文件头部（元数据）
	docCount := 0
	if dc, ok := dbInfo["doc_count"].(float64); ok {
		docCount = int(dc)
	}
	
	header := fmt.Sprintf(`{"db_name":"%s","update_seq":"%v","doc_count":%d,"timestamp":"%s","docs":[`,
		dbName,
		dbInfo["update_seq"],
		docCount,
		time.Now().Format(time.RFC3339),
	)
	
	if _, err := file.WriteString(header); err != nil {
		return 0, err
	}
	
	// 分批导出文档，避免 OOM
	// 每批处理 1000 个文档
	batchSize := 1000
	totalExported := 0
	
	for skip := 0; skip < docCount; skip += batchSize {
		// 获取一批文档
		limit := batchSize
		if skip+limit > docCount {
			limit = docCount - skip
		}
		
		docs, err := bs.exportCouchDBDocsBatch(dbName, skip, limit)
		if err != nil {
			return 0, fmt.Errorf("failed to export batch at skip=%d: %w", skip, err)
		}
		
		// 写入文档
		for i, doc := range docs {
			if totalExported > 0 || i > 0 {
				file.WriteString(",")
			}
			
			// 将文档序列化为 JSON 并写入
			docJSON, err := json.Marshal(doc)
			if err != nil {
				bs.log.Warnf("Failed to marshal doc in %s: %v", dbName, err)
				continue
			}
			
			file.Write(docJSON)
			totalExported++
		}
		
		// 输出进度（每 10 批）
		if (skip/batchSize)%10 == 0 && skip > 0 {
			progress := float64(totalExported) / float64(docCount) * 100
			bs.log.Printf("  Progress: %d/%d docs (%.1f%%)", totalExported, docCount, progress)
		}
	}
	
	// 写入文件尾部
	file.WriteString("]}")
	
	// 获取文件大小
	stat, _ := os.Stat(outputPath)
	
	if totalExported != docCount {
		bs.log.Warnf("Expected %d docs, exported %d docs for %s", docCount, totalExported, dbName)
	}
	
	return stat.Size(), nil
}

// exportCouchDBDocsBatch 分批导出文档
func (bs *BackupScheduler) exportCouchDBDocsBatch(dbName string, skip, limit int) ([]interface{}, error) {
	url := fmt.Sprintf("%s/%s/_all_docs?include_docs=true&skip=%d&limit=%d", 
		bs.couchdbClient.GetURL(), dbName, skip, limit)
	
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	
	bs.setBasicAuth(req)
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}
	
	var result struct {
		Rows []struct {
			Doc interface{} `json:"doc"`
		} `json:"rows"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	docs := make([]interface{}, 0, len(result.Rows))
	for _, row := range result.Rows {
		if row.Doc != nil {
			docs = append(docs, row.Doc)
		}
	}
	
	return docs, nil
}

// setBasicAuth 设置 HTTP Basic Auth
func (bs *BackupScheduler) setBasicAuth(req *http.Request) {
	config := bs.couchdbClient.GetConfig()
	req.SetBasicAuth(config.AdminUser, config.AdminPass)
}

// getPaidUserDatabases 获取付费用户的数据库列表（排除 free 用户）
func (bs *BackupScheduler) getPaidUserDatabases() map[string]bool {
	paidDBs := make(map[string]bool)

	if bs.contentServer == nil {
		bs.log.Warnln("ContentServer is nil, will backup all user databases")
		return paidDBs
	}

	// 获取所有 license
	ns := "License"
	all := bs.contentServer.Repo.AllContent(ns)
	p, ok := bs.contentServer.AllAdminTypes()[ns]
	if !ok {
		bs.log.Warnf("License type not found, will backup all user databases")
		return paidDBs
	}

	paidCount := 0
	freeCount := 0

	// 遍历所有 license，找出付费用户
	for i, v := range all {
		post := p()
		err := json.Unmarshal(v, post)
		if err != nil {
			bs.log.Warnf("Error unmarshalling license %d: %v", i, err)
			continue
		}

		// 类型断言为 *License
		if license, ok := post.(*valueobject.License); ok {
			// 只备份非 free 用户
			if license.Plan != valueobject.PlanFree {
				userDir := license.ToUserDir()
				dbName := fmt.Sprintf("%s%s", bs.couchdbClient.GetConfig().DBPrefix, userDir)
				paidDBs[dbName] = true
				paidCount++
			} else {
				freeCount++
			}
		}
	}

	bs.log.Printf("License analysis: %d paid users, %d free users (free users will be skipped)", paidCount, freeCount)

	return paidDBs
}
