package system

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ============================================================================
// 文件系统实现
// ============================================================================

// FileSystemImpl 文件系统实现
type FileSystemImpl struct {
	baseDir    string
	watchers   map[string]*FileWatcherImpl
	watcherMux sync.RWMutex
}

// NewFileSystem 创建新的文件系统
func NewFileSystem(baseDir string) *FileSystemImpl {
	// 确保基础目录存在
	if baseDir == "" {
		// 使用当前工作目录
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "."
		}
		baseDir = filepath.Join(cwd, "data", "files")
	}

	// 创建目录
	os.MkdirAll(baseDir, 0755)

	return &FileSystemImpl{
		baseDir:  baseDir,
		watchers: make(map[string]*FileWatcherImpl),
	}
}

// ============================================================================
// 文件操作实现
// ============================================================================

// FileRead 读取文件内容
func (fs *FileSystemImpl) FileRead(ctx context.Context, path string) ([]byte, error) {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 获取绝对路径
	absPath := fs.getAbsolutePath(path)

	// 检查文件是否存在
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, &SystemError{
			Code:    "FILE_NOT_FOUND",
			Message: fmt.Sprintf("文件不存在: %s", path),
			Cause:   err,
		}
	}

	// 读取文件
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, &SystemError{
			Code:    "READ_ERROR",
			Message: fmt.Sprintf("读取文件失败: %s", path),
			Cause:   err,
		}
	}

	return data, nil
}

// FileWrite 写入文件内容
func (fs *FileSystemImpl) FileWrite(ctx context.Context, path string, data []byte, append bool) error {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 获取绝对路径
	absPath := fs.getAbsolutePath(path)

	// 确保目录存在
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &SystemError{
			Code:    "DIRECTORY_ERROR",
			Message: fmt.Sprintf("创建目录失败: %s", dir),
			Cause:   err,
		}
	}

	// 写入文件
	var err error
	if append {
		// 追加模式
		file, err := os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return &SystemError{
				Code:    "WRITE_ERROR",
				Message: fmt.Sprintf("打开文件失败: %s", path),
				Cause:   err,
			}
		}
		defer file.Close()

		_, err = file.Write(data)
	} else {
		// 覆盖模式
		err = os.WriteFile(absPath, data, 0644)
	}

	if err != nil {
		return &SystemError{
			Code:    "WRITE_ERROR",
			Message: fmt.Sprintf("写入文件失败: %s", path),
			Cause:   err,
		}
	}

	return nil
}

// FileDelete 删除文件
func (fs *FileSystemImpl) FileDelete(ctx context.Context, path string) error {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 获取绝对路径
	absPath := fs.getAbsolutePath(path)

	// 检查文件是否存在
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return &SystemError{
			Code:    "FILE_NOT_FOUND",
			Message: fmt.Sprintf("文件不存在: %s", path),
			Cause:   err,
		}
	}

	// 删除文件
	if err := os.Remove(absPath); err != nil {
		return &SystemError{
			Code:    "DELETE_ERROR",
			Message: fmt.Sprintf("删除文件失败: %s", path),
			Cause:   err,
		}
	}

	return nil
}

// FileCopy 复制文件
func (fs *FileSystemImpl) FileCopy(ctx context.Context, src, dst string) error {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 获取绝对路径
	srcPath := fs.getAbsolutePath(src)
	dstPath := fs.getAbsolutePath(dst)

	// 检查源文件是否存在
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return &SystemError{
			Code:    "FILE_NOT_FOUND",
			Message: fmt.Sprintf("源文件不存在: %s", src),
			Cause:   err,
		}
	}

	// 确保目标目录存在
	dir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &SystemError{
			Code:    "DIRECTORY_ERROR",
			Message: fmt.Sprintf("创建目录失败: %s", dir),
			Cause:   err,
		}
	}

	// 打开源文件
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return &SystemError{
			Code:    "READ_ERROR",
			Message: fmt.Sprintf("打开源文件失败: %s", src),
			Cause:   err,
		}
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return &SystemError{
			Code:    "WRITE_ERROR",
			Message: fmt.Sprintf("创建目标文件失败: %s", dst),
			Cause:   err,
		}
	}
	defer dstFile.Close()

	// 复制文件内容
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return &SystemError{
			Code:    "COPY_ERROR",
			Message: fmt.Sprintf("复制文件失败: %s -> %s", src, dst),
			Cause:   err,
		}
	}

	return nil
}

// FileMove 移动文件
func (fs *FileSystemImpl) FileMove(ctx context.Context, src, dst string) error {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 获取绝对路径
	srcPath := fs.getAbsolutePath(src)
	dstPath := fs.getAbsolutePath(dst)

	// 检查源文件是否存在
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return &SystemError{
			Code:    "FILE_NOT_FOUND",
			Message: fmt.Sprintf("源文件不存在: %s", src),
			Cause:   err,
		}
	}

	// 确保目标目录存在
	dir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &SystemError{
			Code:    "DIRECTORY_ERROR",
			Message: fmt.Sprintf("创建目录失败: %s", dir),
			Cause:   err,
		}
	}

	// 移动文件
	if err := os.Rename(srcPath, dstPath); err != nil {
		return &SystemError{
			Code:    "MOVE_ERROR",
			Message: fmt.Sprintf("移动文件失败: %s -> %s", src, dst),
			Cause:   err,
		}
	}

	return nil
}

// ============================================================================
// 目录操作实现
// ============================================================================

// DirList 列出目录内容
func (fs *FileSystemImpl) DirList(ctx context.Context, path string, recursive bool) ([]FileInfo, error) {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 获取绝对路径
	absPath := fs.getAbsolutePath(path)

	// 检查目录是否存在
	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return nil, &SystemError{
			Code:    "DIRECTORY_NOT_FOUND",
			Message: fmt.Sprintf("目录不存在: %s", path),
			Cause:   err,
		}
	}
	if !info.IsDir() {
		return nil, &SystemError{
			Code:    "NOT_A_DIRECTORY",
			Message: fmt.Sprintf("不是目录: %s", path),
		}
	}

	var files []FileInfo
	if recursive {
		// 递归列出
		err = filepath.Walk(absPath, func(walkPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// 跳过根目录本身
			if walkPath == absPath {
				return nil
			}

			// 转换为相对路径
			relPath, _ := filepath.Rel(absPath, walkPath)

			files = append(files, FileInfo{
				Name:    info.Name(),
				Path:    relPath,
				Size:    info.Size(),
				Mode:    info.Mode().String(),
				ModTime: info.ModTime(),
				IsDir:   info.IsDir(),
			})

			return nil
		})
	} else {
		// 非递归列出
		entries, err := os.ReadDir(absPath)
		if err != nil {
			return nil, &SystemError{
				Code:    "READ_ERROR",
				Message: fmt.Sprintf("读取目录失败: %s", path),
				Cause:   err,
			}
		}

		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			files = append(files, FileInfo{
				Name:    info.Name(),
				Path:    info.Name(),
				Size:    info.Size(),
				Mode:    info.Mode().String(),
				ModTime: info.ModTime(),
				IsDir:   info.IsDir(),
			})
		}
	}

	if err != nil {
		return nil, &SystemError{
			Code:    "LIST_ERROR",
			Message: fmt.Sprintf("列出目录失败: %s", path),
			Cause:   err,
		}
	}

	return files, nil
}

// DirCreate 创建目录
func (fs *FileSystemImpl) DirCreate(ctx context.Context, path string) error {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 获取绝对路径
	absPath := fs.getAbsolutePath(path)

	// 检查目录是否已存在
	if _, err := os.Stat(absPath); !os.IsNotExist(err) {
		return &SystemError{
			Code:    "ALREADY_EXISTS",
			Message: fmt.Sprintf("目录已存在: %s", path),
		}
	}

	// 创建目录
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return &SystemError{
			Code:    "CREATE_ERROR",
			Message: fmt.Sprintf("创建目录失败: %s", path),
			Cause:   err,
		}
	}

	return nil
}

// DirDelete 删除目录
func (fs *FileSystemImpl) DirDelete(ctx context.Context, path string) error {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 获取绝对路径
	absPath := fs.getAbsolutePath(path)

	// 检查目录是否存在
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return &SystemError{
			Code:    "DIRECTORY_NOT_FOUND",
			Message: fmt.Sprintf("目录不存在: %s", path),
			Cause:   err,
		}
	}

	// 删除目录
	if err := os.RemoveAll(absPath); err != nil {
		return &SystemError{
			Code:    "DELETE_ERROR",
			Message: fmt.Sprintf("删除目录失败: %s", path),
			Cause:   err,
		}
	}

	return nil
}

// ============================================================================
// 文件信息实现
// ============================================================================

// FileExists 检查文件是否存在
func (fs *FileSystemImpl) FileExists(ctx context.Context, path string) (bool, error) {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	// 获取绝对路径
	absPath := fs.getAbsolutePath(path)

	_, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, &SystemError{
			Code:    "STAT_ERROR",
			Message: fmt.Sprintf("检查文件状态失败: %s", path),
			Cause:   err,
		}
	}

	return true, nil
}

// FileInfo 获取文件信息
func (fs *FileSystemImpl) FileInfo(ctx context.Context, path string) (*FileInfo, error) {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 获取绝对路径
	absPath := fs.getAbsolutePath(path)

	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return nil, &SystemError{
			Code:    "FILE_NOT_FOUND",
			Message: fmt.Sprintf("文件不存在: %s", path),
			Cause:   err,
		}
	}
	if err != nil {
		return nil, &SystemError{
			Code:    "STAT_ERROR",
			Message: fmt.Sprintf("获取文件信息失败: %s", path),
			Cause:   err,
		}
	}

	return &FileInfo{
		Name:    info.Name(),
		Path:    path,
		Size:    info.Size(),
		Mode:    info.Mode().String(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}, nil
}

// FileSize 获取文件大小
func (fs *FileSystemImpl) FileSize(ctx context.Context, path string) (int64, error) {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	// 获取绝对路径
	absPath := fs.getAbsolutePath(path)

	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return 0, &SystemError{
			Code:    "FILE_NOT_FOUND",
			Message: fmt.Sprintf("文件不存在: %s", path),
			Cause:   err,
		}
	}
	if err != nil {
		return 0, &SystemError{
			Code:    "STAT_ERROR",
			Message: fmt.Sprintf("获取文件大小失败: %s", path),
			Cause:   err,
		}
	}

	return info.Size(), nil
}

// ============================================================================
// 路径操作实现
// ============================================================================

// PathJoin 连接路径元素
func (fs *FileSystemImpl) PathJoin(elem ...string) string {
	return filepath.Join(elem...)
}

// PathAbs 获取绝对路径
func (fs *FileSystemImpl) PathAbs(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", &SystemError{
			Code:    "PATH_ERROR",
			Message: fmt.Sprintf("获取绝对路径失败: %s", path),
			Cause:   err,
		}
	}
	return absPath, nil
}

// PathBase 获取路径的最后一部分
func (fs *FileSystemImpl) PathBase(path string) string {
	return filepath.Base(path)
}

// PathDir 获取路径的目录部分
func (fs *FileSystemImpl) PathDir(path string) string {
	return filepath.Dir(path)
}

// PathExt 获取路径的扩展名
func (fs *FileSystemImpl) PathExt(path string) string {
	return filepath.Ext(path)
}

// ============================================================================
// 文件监控实现（完整版）
// ============================================================================

// FileWatcherImpl 文件监控实现
type FileWatcherImpl struct {
	id        string
	path      string
	watchType string // "file" or "dir"
	recursive bool
	handler   FileEventHandler
	createdAt time.Time
	events    int
	stopChan  chan struct{}
}

// WatchFile 监控文件
func (fs *FileSystemImpl) WatchFile(ctx context.Context, path string, handler FileEventHandler) (string, error) {
	// 获取绝对路径
	absPath := fs.getAbsolutePath(path)

	// 检查文件是否存在
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", &SystemError{
			Code:    "FILE_NOT_FOUND",
			Message: fmt.Sprintf("文件不存在: %s", path),
			Cause:   err,
		}
	}

	// 生成监控ID
	watchID := fmt.Sprintf("watch_%d_%s", time.Now().UnixNano(), path)

	// 创建监控器
	watcher := &FileWatcherImpl{
		id:        watchID,
		path:      absPath,
		watchType: "file",
		recursive: false,
		handler:   handler,
		createdAt: time.Now(),
		events:    0,
		stopChan:  make(chan struct{}),
	}

	// 保存监控器
	fs.watcherMux.Lock()
	fs.watchers[watchID] = watcher
	fs.watcherMux.Unlock()

	// 启动监控协程
	go fs.watchFileLoop(watcher)

	return watchID, nil
}

// WatchDir 监控目录
func (fs *FileSystemImpl) WatchDir(ctx context.Context, path string, recursive bool, handler FileEventHandler) (string, error) {
	// 获取绝对路径
	absPath := fs.getAbsolutePath(path)

	// 检查目录是否存在
	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return "", &SystemError{
			Code:    "DIRECTORY_NOT_FOUND",
			Message: fmt.Sprintf("目录不存在: %s", path),
			Cause:   err,
		}
	}
	if !info.IsDir() {
		return "", &SystemError{
			Code:    "NOT_A_DIRECTORY",
			Message: fmt.Sprintf("不是目录: %s", path),
		}
	}

	// 生成监控ID
	watchID := fmt.Sprintf("watch_%d_%s", time.Now().UnixNano(), path)

	// 创建监控器
	watcher := &FileWatcherImpl{
		id:        watchID,
		path:      absPath,
		watchType: "dir",
		recursive: recursive,
		handler:   handler,
		createdAt: time.Now(),
		events:    0,
		stopChan:  make(chan struct{}),
	}

	// 保存监控器
	fs.watcherMux.Lock()
	fs.watchers[watchID] = watcher
	fs.watcherMux.Unlock()

	// 启动监控协程
	go fs.watchDirLoop(watcher)

	return watchID, nil
}

// Unwatch 取消监控
func (fs *FileSystemImpl) Unwatch(watchID string) error {
	fs.watcherMux.Lock()
	defer fs.watcherMux.Unlock()

	watcher, exists := fs.watchers[watchID]
	if !exists {
		return &SystemError{
			Code:    "WATCH_NOT_FOUND",
			Message: fmt.Sprintf("监控不存在: %s", watchID),
		}
	}

	// 停止监控
	close(watcher.stopChan)

	// 移除监控器
	delete(fs.watchers, watchID)

	return nil
}

// ListWatches 列出所有监控
func (fs *FileSystemImpl) ListWatches() []WatchInfo {
	fs.watcherMux.RLock()
	defer fs.watcherMux.RUnlock()

	watches := make([]WatchInfo, 0, len(fs.watchers))
	for _, watcher := range fs.watchers {
		watches = append(watches, WatchInfo{
			ID:        watcher.id,
			Path:      watcher.path,
			Type:      watcher.watchType,
			Recursive: watcher.recursive,
			CreatedAt: watcher.createdAt,
			Events:    watcher.events,
		})
	}

	return watches
}

// GetWatchStats 获取监控统计
func (fs *FileSystemImpl) GetWatchStats(watchID string) (*WatchStats, error) {
	fs.watcherMux.RLock()
	defer fs.watcherMux.RUnlock()

	watcher, exists := fs.watchers[watchID]
	if !exists {
		return nil, &SystemError{
			Code:    "WATCH_NOT_FOUND",
			Message: fmt.Sprintf("监控不存在: %s", watchID),
		}
	}

	return &WatchStats{
		EventsTotal:   watcher.events,
		EventsLastMin: 0,                            // 简化实现
		LastEvent:     time.Now().Add(-time.Minute), // 简化实现
		ErrorCount:    0,
	}, nil
}

// watchFileLoop 文件监控循环
func (fs *FileSystemImpl) watchFileLoop(watcher *FileWatcherImpl) {
	// 记录初始文件状态
	lastModTime := time.Time{}
	lastSize := int64(0)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 检查文件状态
			info, err := os.Stat(watcher.path)
			if err != nil {
				// 文件可能被删除
				if os.IsNotExist(err) {
					watcher.events++
					watcher.handler(&FileEvent{
						Type:      FileEventRemove,
						Path:      watcher.path,
						Timestamp: time.Now(),
					})
				}
				continue
			}

			// 检查文件是否被修改
			if !lastModTime.IsZero() && (info.ModTime() != lastModTime || info.Size() != lastSize) {
				watcher.events++
				watcher.handler(&FileEvent{
					Type:      FileEventWrite,
					Path:      watcher.path,
					Timestamp: time.Now(),
					Size:      info.Size(),
				})
			}

			// 更新状态
			lastModTime = info.ModTime()
			lastSize = info.Size()

		case <-watcher.stopChan:
			return
		}
	}
}

// watchDirLoop 目录监控循环
func (fs *FileSystemImpl) watchDirLoop(watcher *FileWatcherImpl) {
	// 记录初始目录状态
	lastFiles := make(map[string]os.FileInfo)

	// 获取初始文件列表
	fs.collectFiles(watcher.path, watcher.recursive, lastFiles)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 获取当前文件列表
			currentFiles := make(map[string]os.FileInfo)
			fs.collectFiles(watcher.path, watcher.recursive, currentFiles)

			// 检测新文件
			for path, info := range currentFiles {
				if _, exists := lastFiles[path]; !exists {
					watcher.events++
					watcher.handler(&FileEvent{
						Type:      FileEventCreate,
						Path:      path,
						Timestamp: time.Now(),
						Size:      info.Size(),
					})
				}
			}

			// 检测删除的文件
			for path := range lastFiles {
				if _, exists := currentFiles[path]; !exists {
					watcher.events++
					watcher.handler(&FileEvent{
						Type:      FileEventRemove,
						Path:      path,
						Timestamp: time.Now(),
					})
				}
			}

			// 更新文件列表
			lastFiles = currentFiles

		case <-watcher.stopChan:
			return
		}
	}
}

// collectFiles 收集目录下的所有文件
func (fs *FileSystemImpl) collectFiles(rootPath string, recursive bool, files map[string]os.FileInfo) {
	if recursive {
		filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			files[path] = info
			return nil
		})
	} else {
		entries, err := os.ReadDir(rootPath)
		if err != nil {
			return
		}
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			files[filepath.Join(rootPath, info.Name())] = info
		}
	}
}

// ============================================================================
// 辅助方法
// ============================================================================

// getAbsolutePath 获取绝对路径
func (fs *FileSystemImpl) getAbsolutePath(path string) string {
	// 如果路径已经是绝对路径，直接返回
	if filepath.IsAbs(path) {
		return path
	}

	// 否则相对于基础目录
	return filepath.Join(fs.baseDir, path)
}
