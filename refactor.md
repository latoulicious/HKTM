# **🎯 New SPEC: Pragmatic Robust Audio Pipeline**

## **📋 Requirements (Simplified)**

### **Core Requirements**
1. **Robust Audio Streaming** - Handle network issues, FFmpeg crashes, Discord disconnects
2. **Simple Database Caching** - Store external API data to reduce hits (UMA game data, etc.)
3. **Fault Tolerance** - Basic error recovery without enterprise complexity
4. **Easy to Build** - Simple patterns, minimal interfaces, clear structure

### **Non-Requirements (What to Remove)**
- ❌ Enterprise monitoring systems
- ❌ Complex recovery learning algorithms  
- ❌ Performance profiling and analytics
- ❌ Resource management pools
- ❌ Health check schedulers
- ❌ Alert management systems

## **🏗️ Architecture (Simplified)**

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Discord Bot   │───▶│  Audio Pipeline │───▶│   Database      │
│   Commands      │    │   (Simple)      │    │   (GORM)        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │   FFmpeg +      │
                       │   Discord API   │
                       └─────────────────┘
```

## **📦 Package Structure (Simplified)**

```
pkg/
├── audio/
│   ├── pipeline.go      # Main audio pipeline (200-300 lines)
│   ├── ffmpeg.go        # FFmpeg process management (100 lines)
│   └── discord.go       # Discord voice streaming (100 lines)
├── database/
│   ├── models.go        # GORM models (50 lines)
│   ├── repository.go    # Simple repository pattern (100 lines)
│   └── cache.go         # API data caching (50 lines)
└── common/
    ├── errors.go        # Simple error types (20 lines)
    └── config.go        # Basic configuration (50 lines)
```

## **🔧 Implementation Spec**

### **1. Simple Audio Pipeline**

```go
// pkg/audio/pipeline.go (~200 lines)
type AudioPipeline struct {
    voiceConn *discordgo.VoiceConnection
    ffmpegCmd *exec.Cmd
    isPlaying bool
    mu        sync.RWMutex
    
    // Simple error handling
    errorChan chan error
    logger    *log.Logger
}

func (ap *AudioPipeline) Play(url string) error {
    // 1. Download with yt-dlp
    // 2. Process with FFmpeg
    // 3. Stream to Discord
    // 4. Handle basic errors
}

func (ap *AudioPipeline) Stop() error {
    // Clean shutdown
}

func (ap *AudioPipeline) Pause() error {
    // Simple pause/resume
}
```

### **2. Simple Database (Your Pattern)**

```go
// pkg/database/repository.go (~100 lines)
type BaseRepository struct {
    DB *gorm.DB
}

func (r *BaseRepository) Create(model interface{}, tx *gorm.DB) error {
    db := r.DB
    if tx != nil {
        db = tx
    }
    return db.Create(model).Error
}

type UMARepository struct {
    BaseRepository
}

func (r *UMARepository) CacheCharacterSearch(query string, data string, ttl time.Duration) error {
    cache := &models.CharacterSearchCache{
        Query: query,
        Data: data,
        ExpiresAt: time.Now().Add(ttl),
    }
    return r.Create(cache, nil)
}
```

### **3. Simple Models**

```go
// pkg/database/models.go (~50 lines)
type CharacterSearchCache struct {
    ID        uuid.UUID `gorm:"primaryKey"`
    Query     string    `gorm:"uniqueIndex;not null"`
    Data      string    `gorm:"type:text;not null"`
    ExpiresAt time.Time `gorm:"index;not null"`
    CreatedAt time.Time `gorm:"autoCreateTime"`
}

type PipelineMetric struct {
    ID         uuid.UUID `gorm:"primaryKey"`
    PipelineID string    `gorm:"index;not null"`
    MetricName string    `gorm:"index;not null"`
    Value      float64   `gorm:"not null"`
    Timestamp  time.Time `gorm:"index;not null"`
}
```

### **4. Simple Error Handling**

```go
// pkg/common/errors.go (~20 lines)
var (
    ErrStreamNotFound = errors.New("stream not found")
    ErrFFmpegFailed   = errors.New("ffmpeg process failed")
    ErrDiscordFailed  = errors.New("discord connection failed")
    ErrNetworkFailed  = errors.New("network error")
)
```

## **🚀 Implementation Plan**

### **Phase 1: Core Audio (Week 1)**
- [ ] Create simple `AudioPipeline` struct
- [ ] Implement basic FFmpeg integration
- [ ] Add Discord voice streaming
- [ ] Basic error handling and recovery

### **Phase 2: Database (Week 1)**
- [ ] Create GORM models for caching
- [ ] Implement simple repository pattern
- [ ] Add UMA cache functionality
- [ ] Basic metrics storage

### **Phase 3: Integration (Week 2)**
- [ ] Integrate with existing Discord commands
- [ ] Add simple logging
- [ ] Basic configuration management
- [ ] Write tests

### **Phase 4: Robustness (Week 2)**
- [ ] Add retry logic for network issues
- [ ] Implement graceful shutdown
- [ ] Add basic health checks
- [ ] Error recovery strategies

## **📊 Complexity Comparison**

| Component | Current | Simplified | Reduction |
|-----------|---------|------------|-----------|
| **Audio Pipeline** | 16,000+ lines | 500 lines | **97%** |
| **Database** | 3,000+ lines | 300 lines | **90%** |
| **Interfaces** | 30+ interfaces | 5 interfaces | **83%** |
| **Files** | 38+ files | 8 files | **79%** |

## **🎯 Key Improvements**

### **Keep the Good Parts:**
- ✅ **Robust error handling** (simplified)
- ✅ **Database caching** (your simple pattern)
- ✅ **FFmpeg integration** (simplified)
- ✅ **Discord voice streaming** (simplified)

### **Remove the Complexity:**
- ❌ **Enterprise monitoring** → Simple logging
- ❌ **Complex recovery** → Basic retry logic
- ❌ **Performance profiling** → Basic metrics
- ❌ **Resource pools** → Simple resource management
- ❌ **Health schedulers** → Basic health checks

## **�� Migration Strategy**

### **Option 1: Gradual Migration (Recommended)**
1. **Keep existing pipeline** as reference
2. **Build simple pipeline** alongside it
3. **Migrate commands** one by one
4. **Remove complex parts** after migration

### **Option 2: Clean Slate**
1. **Backup current code** for reference
2. **Build simple pipeline** from scratch
3. **Migrate all at once**

## **💡 Benefits of Simplified Approach**

1. **Easier to understand** - Clear, simple code
2. **Easier to debug** - Fewer moving parts
3. **Easier to maintain** - Less complexity
4. **Easier to extend** - Simple patterns
5. **Still robust** - Handles real-world issues
6. **Better for side project** - Focus on features, not infrastructure

## **🎯 Next Steps**

1. **Start with simple audio pipeline** - 200-300 lines
2. **Use your database pattern** - Simple GORM repositories
3. **Add basic robustness** - Retry logic, error handling
4. **Integrate with commands** - Replace complex pipeline
5. **Remove over-engineered parts** - Keep only what you need

This gives you **robust, fault-tolerant audio** without the enterprise complexity. You'll have a system that's:
- ✅ **Easy to build and maintain**
- ✅ **Handles real-world issues** 
- ✅ **Uses your preferred patterns**
- ✅ **Still impressive for portfolio**
