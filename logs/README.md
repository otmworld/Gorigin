# Log File Write.

## All right global use way eg:

```go
   // create global values.
   var l LogData
   // initialized global values.
   func init() {
        l = LogInit(&LogStruct{})
   }
   // write log data in file.
   func logWrite() {
        l.WriteError("message")
   }
```

## Struct Definition

```go
// It can be null.
type LogStruct struct {
        // if true, mean log data will put in cache first, than cache full put in file.
        // if false, mean log data put in file as first time.
        // when LogStruct was null, Cache default true.
        Cache bool
        // cache save size (byte), default 1024*1024 byte
        CacheSize int
        // log time format, default "2006-01-02 15:04:05"
        TimeFormat string
        // log file pre name. default "log".
        FileName string
        // file save path.
        FilePath string
        // log save level. default error level.
        Level LogLevel
        // how long about file create.
        // when FileName was null, mean pre name eq TimeDay.
        FileTime LogTime
        // whether create dir to save log file.
        Dir bool
}
```

### Use Panic interfase will make log file close