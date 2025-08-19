# URL Refresh Detection Implementation Summary

## Task: Add basic URL refresh detection

### Requirements Implemented:

#### Requirement 8.1: Track URL's expected expiration time
✅ **IMPLEMENTED**: 
- Added `urlExpiry`, `urlStartTime`, `originalURL`, `streamURL` fields to `FFmpegProcessor`
- `refreshStreamURL()` method tracks URL expiry time (5 minutes for YouTube URLs)
- URL expiry is logged and monitored throughout the streaming process

#### Requirement 8.2: Proactively fetch fresh URL when close to expiring  
✅ **IMPLEMENTED**:
- `startURLRefreshTimer()` sets up proactive refresh 1 minute before expiry
- `handleURLRefresh()` performs background URL refresh with retry logic
- `HandleStreamFailureWithRefresh()` provides on-demand URL refresh for stream failures
- Fresh URLs are obtained using yt-dlp and seamlessly integrated into the pipeline

#### Requirement 6.1: Retry 2-3 times with short delays when network fails
✅ **IMPLEMENTED**:
- `startStreamWithRetry()` implements retry logic with 3 attempts max
- Delay pattern: 2s, 5s, 10s between retries
- `handleURLRefresh()` includes retry logic for URL refresh failures (3 attempts)
- Pipeline controller integrates URL refresh into error handling for first 2 attempts

#### Requirement 6.2: Show error and skip song when yt-dlp can't get URL
✅ **IMPLEMENTED**:
- `DetectStreamFailure()` identifies URL expiry patterns (403, 404, connection refused, etc.)
- Comprehensive error logging with context fields for debugging
- After max retries, errors are properly logged and propagated to skip to next song
- Clear error messages distinguish between URL refresh failures and other errors

### Key Implementation Details:

1. **Simple URL Expiry Detection**: 
   - Pattern matching for common URL expiry errors (403, 404, connection refused, etc.)
   - Conservative 5-minute TTL assumption for YouTube URLs

2. **Automatic Retry with Fresh URL**:
   - Integrated into existing retry logic in `startStreamWithRetry()`
   - Pipeline controller detects URL expiry errors and triggers refresh
   - Seamless fallback to fresh URLs during retries

3. **Keep it Simple**:
   - Uses existing yt-dlp `--get-url` command to get fresh URLs
   - Minimal changes to existing StreamProcessor interface
   - Reuses existing error handling and logging infrastructure

4. **Proactive Background Refresh**:
   - Timer-based refresh 1 minute before expiry
   - Background refresh doesn't interrupt active streaming
   - Proper cleanup of timers during shutdown

### Files Modified:

1. **pkg/audio/interfaces.go**: Added URL refresh methods to StreamProcessor interface
2. **pkg/audio/ffmpeg.go**: Core URL refresh detection and management logic
3. **pkg/audio/pipeline.go**: Integration with existing error handling
4. **test/audio/url_refresh_test.go**: Unit tests for URL expiry detection

### Testing:

- ✅ Unit tests verify URL expiry pattern detection
- ✅ Build passes without compilation errors  
- ✅ Integration with existing pipeline controller
- ✅ Proper error handling and logging

The implementation successfully adds basic URL refresh detection to the existing StreamProcessor while maintaining simplicity and integrating seamlessly with the current architecture.