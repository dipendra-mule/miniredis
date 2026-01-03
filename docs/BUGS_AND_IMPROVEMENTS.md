# Bugs and Improvements Report

This document lists all identified bugs, potential issues, and improvement opportunities in the MiniRedis codebase.

## Critical Bugs

### 1. Race Condition in `bgsave` (handler.go:240-265)
**Location**: `handler.go:240-265`  
**Severity**: High  
**Issue**: The `state.bgsaveRunning` flag and `state.dbCopy` are accessed/modified without mutex protection, leading to race conditions when multiple clients call `BGSAVE` concurrently.

**Problem**:
```go
if state.bgsaveRunning {  // Not protected by mutex
    return &Resp{...}
}
state.bgsaveRunning = true  // Race condition here
state.dbCopy = cp
```

**Fix**: Add mutex protection to `AppState` struct for `bgsaveRunning` and `dbCopy`.

---

### 2. Race Condition in AOF Rewrite (aof.go:51-83)
**Location**: `aof.go:51-83`  
**Severity**: High  
**Issue**: The `aof.w` writer is reassigned without mutex protection, and commands written during rewrite are lost (they go to a buffer that's discarded).

**Problem**:
- `aof.w` is reassigned to a buffer during rewrite
- Commands written during rewrite go to buffer but buffer is never appended back
- No mutex protection on `aof.w` reassignment

**Fix**: 
- Add mutex to protect `aof.w`
- Append buffered commands back to file after rewrite completes
- Or use a temporary file and atomic rename

---

### 3. Thread Safety Issue in RDB Tracker (rdb.go:29, 50-54)
**Location**: `rdb.go:29, 50-54`  
**Severity**: High  
**Issue**: Global `trackers` slice and `IncrRDBTracker()` function are not thread-safe.

**Problem**:
```go
var trackers = []*SnapshotTracker{}  // Global, not protected

func IncrRDBTracker() {
    for _, t := range trackers {  // Reading without lock
        t.keys++  // Writing without lock
    }
}
```

**Fix**: Add mutex protection for trackers slice access and key increments.

---

### 4. Missing Bounds Check in Command Parsing (handler.go:39)
**Location**: `handler.go:39`  
**Severity**: High  
**Issue**: Accessing `r.arr[0]` without checking if array is empty will cause panic.

**Problem**:
```go
cmd := r.arr[0].bulk  // Panic if r.arr is empty
```

**Fix**: Add bounds check before accessing `r.arr[0]`.

---

### 5. Double Close in SyncRDB (rdb.go:114-129)
**Location**: `rdb.go:119, 122`  
**Severity**: Medium  
**Issue**: File is closed twice - once on error path and once in defer.

**Problem**:
```go
f, err := os.Open(fp)
if err != nil {
    log.Println("error opening rdb file: ", err)
    f.Close()  // f is nil, this will panic
    return
}
defer f.Close()  // This will also execute
```

**Fix**: Remove `f.Close()` from error path since `f` is nil.

---

### 6. Missing Null Check in bgwriteaof (handler.go:395-403)
**Location**: `handler.go:395-403`  
**Severity**: High  
**Issue**: `bgwriteaof` doesn't check if AOF is enabled or if `state.aof` is nil, causing panic.

**Problem**:
```go
func bgwriteaof(...) {
    go func() {
        // ...
        state.aof.Rewrite(cp)  // Panic if aof is nil
    }()
}
```

**Fix**: Add nil check and AOF enabled check before calling Rewrite.

---

## Medium Severity Bugs

### 7. Race Condition in GET Command (handler.go:147-154)
**Location**: `handler.go:147-154`  
**Severity**: Medium  
**Issue**: Key expiration check has a race condition - key could expire between RLock release and Lock acquisition.

**Problem**:
```go
DB.mu.RLock()
val, ok := DB.store[args[0].bulk]
DB.mu.RUnlock()  // Lock released

if val.Exp.Unix() != UNIX_TS_EPOCH && time.Until(val.Exp).Seconds() <= 0 {
    DB.mu.Lock()  // Key might have been deleted/modified here
    DB.Delete(args[0].bulk)
    DB.mu.Unlock()
}
```

**Fix**: Keep lock held during expiration check, or re-check after acquiring write lock.

---

### 8. Incorrect Condition in SET Handler (handler.go:112)
**Location**: `handler.go:112`  
**Severity**: Medium  
**Issue**: Condition `len(state.conf.rdb) >= 0` is always true.

**Problem**:
```go
if len(state.conf.rdb) >= 0 {  // Always true
    IncrRDBTracker()
}
```

**Fix**: Should be `len(state.conf.rdb) > 0`.

---

### 9. Missing Error Handling in Writer (writer.go:46)
**Location**: `writer.go:46`  
**Severity**: Medium  
**Issue**: `Write()` doesn't check for errors from underlying writer.

**Problem**:
```go
func (w *Writer) Write(r *Resp) {
    reply := w.Deserialize(r)
    w.writer.Write([]byte(reply))  // Error ignored
}
```

**Fix**: Check and return error from Write operation.

---

### 10. Unsafe Type Assertion in Flush (writer.go:50)
**Location**: `writer.go:50`  
**Severity**: Medium  
**Issue**: Type assertion could panic if writer is not `*bufio.Writer`.

**Problem**:
```go
func (w *Writer) Flush() {
    w.writer.(*bufio.Writer).Flush()  // Panic if not bufio.Writer
}
```

**Fix**: Use type assertion with ok check or store bufio.Writer directly.

---

### 11. parseRespArr Doesn't Reset Array (resp.go:44-64)
**Location**: `resp.go:44-64`  
**Severity**: Medium  
**Issue**: `parseRespArr` doesn't reset `r.arr` before parsing, causing accumulation of old data on repeated calls.

**Problem**:
```go
func (r *Resp) parseRespArr(reader io.Reader) error {
    // r.arr is not reset, so old elements accumulate
    for range arrLen {
        bulk := r.parseBulkStr(rd)
        r.arr = append(r.arr, bulk)  // Appends to existing array
    }
}
```

**Fix**: Reset `r.arr = nil` or `r.arr = []Resp{}` at the start.

---

### 12. Missing Bounds Check in parseRespArr (resp.go:51)
**Location**: `resp.go:51`  
**Severity**: Medium  
**Issue**: Accessing `line[0]` without checking if line is empty.

**Problem**:
```go
if line[0] != '*' {  // Panic if line is empty
    return errors.New("expcted array")
}
```

**Fix**: Check `len(line) > 0` before accessing `line[0]`.

---

### 13. parseBulkStr Doesn't Handle Null Bulk Strings (resp.go:67-91)
**Location**: `resp.go:67-91`  
**Severity**: Medium  
**Issue**: Doesn't handle negative length (null bulk string: `$-1\r\n`).

**Problem**: RESP protocol allows `$-1\r\n` for null bulk strings, but code doesn't handle this case.

**Fix**: Check for negative length and return appropriate null Resp.

---

### 14. Incorrect Log Message in RDB Tracker (rdb.go:40)
**Location**: `rdb.go:40`  
**Severity**: Low  
**Issue**: Log message uses wrong variable name.

**Problem**:
```go
log.Printf("keys changed: %d - keys req to change: %d", tracker.keys, tracker.rdb.Secs)
// Should be tracker.rdb.KeysChanged, not Secs
```

**Fix**: Use `tracker.rdb.KeysChanged` instead of `tracker.rdb.Secs`.

---

### 15. Missing Space in Error Message (handler.go:101)
**Location**: `handler.go:101`  
**Severity**: Low  
**Issue**: Missing space between "ERR" and error message.

**Problem**:
```go
err: "ERR" + err.Error()  // Results in "ERRmaximum memory reached"
```

**Fix**: Should be `"ERR " + err.Error()`.

---

### 16. KEYS Command Doesn't Handle Empty Pattern (handler.go:199-206)
**Location**: `handler.go:199-206`  
**Severity**: Medium  
**Issue**: KEYS command doesn't handle case when no pattern is provided (empty args).

**Problem**:
```go
if len(args) > 1 {  // Only checks upper bound
    return &Resp{...}
}
pattern := args[0].bulk  // Panic if args is empty
```

**Fix**: Check `len(args) == 0` and return error or use default pattern.

---

### 17. Transaction Error Handling (handler.go:427-445)
**Location**: `handler.go:427-445`  
**Severity**: Medium  
**Issue**: Transactions don't handle errors properly - if one command fails, others still execute. Also, transactions don't write to AOF atomically.

**Problem**:
- No rollback mechanism
- AOF writes happen per command, not per transaction
- If transaction fails partway, partial state is persisted

**Fix**: 
- Implement proper transaction rollback
- Write entire transaction to AOF atomically on EXEC
- Consider using WATCH/UNWATCH for optimistic locking

---

## Code Quality Issues

### 18. Typos in Code
**Locations**:
- `main.go:43, 55`: "accepeted" should be "accepted"
- `resp.go:52`: "expcted" should be "expected"
- `writer.go:38`: "typ" should be "type"

**Severity**: Low  
**Fix**: Correct spelling.

---

### 19. Commented Out Code (main.go:51)
**Location**: `main.go:51`  
**Severity**: Low  
**Issue**: `wg.Wait()` is commented out, making WaitGroup useless.

**Problem**: WaitGroup is created but never waited on, so it serves no purpose.

**Fix**: Either remove WaitGroup or implement proper graceful shutdown with WaitGroup.

---

### 20. Magic Number (main.go:12)
**Location**: `main.go:12`  
**Severity**: Low  
**Issue**: `UNIX_TS_EPOCH` constant value is a magic number without explanation.

**Fix**: Add comment explaining why this specific value is used.

---

### 21. Case Sensitivity Inconsistency (handler.go:32-36)
**Location**: `handler.go:32-36`  
**Severity**: Low  
**Issue**: Command matching is case-sensitive, but SafeCMDs includes both "AUTH" and "auth".

**Problem**: Commands are matched case-sensitively, so "auth" won't match "AUTH" handler.

**Fix**: Normalize commands to uppercase before matching, or make matching case-insensitive.

---

### 22. Inefficient Checksum Calculation (rdb.go:81-109)
**Location**: `rdb.go:81-109`  
**Severity**: Low  
**Issue**: Checksum is calculated twice - once on buffer, then again on file after writing.

**Problem**: Inefficient - could calculate checksum once after writing.

**Fix**: Calculate checksum only once after writing to file.

---

### 23. Database Locking Inconsistency (db.go:29-49, handler.go:94)
**Location**: `db.go:29-49, handler.go:94`  
**Severity**: Medium  
**Issue**: `DB.Set()` is called with lock already held, but `Set()` doesn't document this requirement.

**Problem**: Locking responsibility is unclear - handler locks, but Set() might expect to lock itself.

**Fix**: Either remove lock from handler and add to Set(), or document that Set() requires lock to be held.

---

### 24. Memory Calculation Accuracy (db.go:70-76)
**Location**: `db.go:70-76`  
**Severity**: Low  
**Issue**: Memory calculation is approximate and might not match actual memory usage.

**Problem**: Uses fixed header sizes that might not match Go's actual memory layout.

**Fix**: Consider using `runtime.MemStats` or more accurate calculation methods.

---

### 25. No Graceful Shutdown (main.go:14-52)
**Location**: `main.go:14-52`  
**Severity**: Medium  
**Issue**: Server doesn't handle shutdown signals (SIGINT, SIGTERM) gracefully.

**Problem**: 
- No way to stop server cleanly
- AOF goroutine runs forever
- Connections are not closed gracefully

**Fix**: Implement signal handling and graceful shutdown.

---

### 26. AOF Sync Goroutine Never Stops (main.go:95-103)
**Location**: `main.go:95-103`  
**Severity**: Medium  
**Issue**: Goroutine for AOF flushing runs forever with no way to stop it.

**Problem**: No shutdown mechanism for the ticker goroutine.

**Fix**: Add context cancellation or shutdown channel.

---

### 27. Partial Initialization on Error (aof.go:18-30)
**Location**: `aof.go:18-30`  
**Severity**: Medium  
**Issue**: `NewAof()` returns partially initialized struct on error (aof.w is nil).

**Problem**: Callers might not check if aof.w is nil before using it.

**Fix**: Return error instead of partial struct, or ensure all fields are initialized.

---

### 28. Config Parsing Issues (conf.go:76-127)
**Location**: `conf.go:76-127`  
**Severity**: Medium  
**Issues**:
- Doesn't handle comments (lines starting with `#`)
- Doesn't handle empty lines
- Doesn't handle quoted values with spaces
- `strings.Split()` breaks on values with spaces
- No validation that array indices exist before accessing
- `os.MkdirAll()` error is ignored

**Fix**: 
- Skip comment lines and empty lines
- Support quoted values
- Validate array bounds
- Handle directory creation errors

---

### 29. No RDB Checksum Validation on Load (rdb.go:114-129)
**Location**: `rdb.go:114-129`  
**Severity**: Medium  
**Issue**: `SyncRDB()` doesn't validate checksum of loaded RDB file.

**Problem**: Corrupted RDB files might be loaded without detection.

**Fix**: Validate checksum after loading RDB file.

---

### 30. Checksum Calculation Bug (rdb.go:81-85)
**Location**: `rdb.go:81-85`  
**Severity**: Medium  
**Issue**: Checksum is calculated on buffer, but buffer position might be wrong after encoding.

**Problem**: After `gob.Encode()`, buffer position is at end, so `Hash(&buf)` reads from wrong position.

**Fix**: Reset buffer position before hashing, or hash the bytes directly.

---

### 31. Client Without Connection in AOF Sync (aof.go:46)
**Location**: `aof.go:46`  
**Severity**: Low  
**Issue**: Uses blank `Client{}` without connection for replaying AOF.

**Problem**: Client struct expects connection, but it's not used in `set()` handler, so this might be okay but is confusing.

**Fix**: Either document why connection is not needed, or create proper client.

---

### 32. Global Database Variable (db.go:63)
**Location**: `db.go:63`  
**Severity**: Low  
**Issue**: Global `DB` variable makes testing difficult and prevents multiple instances.

**Fix**: Consider dependency injection or making DB part of AppState.

---

### 33. No Connection Timeout Handling (main.go:54-66)
**Location**: `main.go:54-66`  
**Severity**: Low  
**Issue**: No read/write timeouts on connections, could lead to hanging connections.

**Fix**: Set read/write deadlines on connections.

---

### 34. Inefficient String Contains (utils.go:3-11)
**Location**: `utils.go:3-11`  
**Severity**: Low  
**Issue**: Custom `contains()` function when `strings.Contains()` or `slices.Contains()` could be used.

**Fix**: Use standard library function or document why custom implementation is needed.

---

## Missing Features / Improvements

### 35. No Error Return from parseBulkStr (resp.go:67-91)
**Location**: `resp.go:67-91`  
**Severity**: Low  
**Issue**: Returns empty Resp on error, caller can't distinguish between error and empty string.

**Fix**: Return error as second return value.

---

### 36. No Validation of RDB File Existence (rdb.go:114-129)
**Location**: `rdb.go:114-129`  
**Severity**: Low  
**Issue**: Doesn't check if RDB file exists before trying to open it.

**Fix**: Check file existence or handle "file not found" error gracefully.

---

### 37. Memory Limit Check Uses >= Instead of > (db.go:38)
**Location**: `db.go:38`  
**Severity**: Low  
**Issue**: Uses `>=` which might allow exactly at limit, but semantics unclear.

**Fix**: Clarify intended behavior and use appropriate comparison.

---

### 38. No Expiration Cleanup Background Task
**Location**: N/A  
**Severity**: Medium  
**Issue**: Expired keys are only removed when accessed (lazy deletion). No background task to clean up expired keys.

**Fix**: Add background goroutine to periodically clean up expired keys.

---

### 39. Transaction Commands Not Validated (handler.go:60-68)
**Location**: `handler.go:60-68`  
**Severity**: Low  
**Issue**: Commands are queued in transactions without validation.

**Fix**: Validate commands can be executed before queuing (or document that validation happens on EXEC).

---

### 40. No Maximum Command Size Limit
**Location**: `resp.go`, `handler.go`  
**Severity**: Low  
**Issue**: No limit on command or value size, could lead to memory exhaustion.

**Fix**: Add configurable limits on command and value sizes.

---

## Summary

**Critical Bugs**: 6  
**Medium Severity**: 11  
**Low Severity**: 23  

**Total Issues Found**: 40

### Priority Fixes
1. Race conditions in bgsave, AOF rewrite, and RDB tracker
2. Missing bounds checks that could cause panics
3. Thread safety issues
4. Error handling improvements
5. Graceful shutdown implementation

