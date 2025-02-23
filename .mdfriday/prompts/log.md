# Log Best Practices

## Logging Levels

- **Error**: For errors that prevent the application from functioning correctly
- **Warn**: For potentially harmful situations that don't stop execution
- **Info**: For general operational information
- **Debug**: For detailed information useful during development

## Structured Logging

Use structured logging with fields for better searchability and context:

### 1. Field Management

Create a dedicated fields type to manage structured log fields:

```go
type LogFields struct {
    fields logg.Fields
}

func (f *LogFields) Fields() logg.Fields {
    return f.fields
}

func (f *LogFields) addField(name string, value interface{}) {
    f.fields = append(f.fields, logg.Field{Name: name, Value: value})
}
```

### 2. Common Fields

Always include common contextual fields for operations:

```go
func newLogFields(operation string) *LogFields {
    return &LogFields{
        fields: logg.Fields{
            {Name: "sessionID", Value: sessionID},
            {Name: "operation", Value: operation},
            {Name: "timestamp", Value: time.Now()},
        },
    }
}
```

### 3. Operation Timing

Track operation duration using defer:

```go
func SomeOperation() {
    operationLog := logger.Info()
    defer loggers.TimeTrackf(operationLog, time.Now(), nil, "")
    
    // operation code...
}
```

### 4. Error Logging

Include error context and maintain error chain:

```go
if err != nil {
    logger.Error().
        WithFields(fields).
        WithError(err).
        Logf("Failed to process request")
    return errors.Wrap(err, "operation failed")
}
```

### 5. Operation Progress

Log the start and completion of significant operations:

```go
func ProcessItem(item string) error {
    log := logger.Info()
    fields := newLogFields("process_item")
    fields.addField("item", item)
    
    log.WithFields(fields).Logf("Starting item processing")
    
    // Process item...
    
    log.WithFields(fields).Logf("Item processing completed")
    return nil
}
```

### 6. Contextual Information

Include relevant context in log fields:

```go
fields.addFields(
    logg.Field{Name: "path", Value: path},
    logg.Field{Name: "size", Value: size},
    logg.Field{Name: "mode", Value: mode},
    logg.Field{Name: "user", Value: username},
)
```

## Best Practices

1. **Consistent Structure**: Use consistent field names across the application

2. **Operation Context**: Always include operation name and session/request ID

3. **Error Chain**: Maintain error context when wrapping errors

4. **Performance**: Use defer for timing tracking

5. **Granular Logging**: Log at appropriate levels:
   - Info: Operation start/completion
   - Error: Operation failures with context
   - Debug: Detailed progress information
   - Warn: Potential issues that don't stop execution

6. **Clean Context**: Clean up any resources or context after logging

## Example Implementation

```go
type Service struct {
    logger loggers.Logger
}

func (s *Service) ProcessRequest(ctx context.Context, req Request) error {
    processLog := s.logger.Info()
    defer loggers.TimeTrackf(processLog, time.Now(), nil, "")

    fields := &LogFields{
        fields: logg.Fields{
            {Name: "requestID", Value: req.ID},
            {Name: "operation", Value: "process_request"},
            {Name: "user", Value: req.UserID},
        },
    }

    processLog.WithFields(fields).Logf("Starting request processing")

    if err := s.validate(req); err != nil {
        processLog.WithFields(fields).
            WithError(err).
            Logf("Request validation failed")
        return errors.Wrap(err, "invalid request")
    }

    result, err := s.process(req)
    if err != nil {
        processLog.WithFields(fields).
            WithError(err).
            Logf("Request processing failed")
        return errors.Wrap(err, "processing error")
    }

    fields.addField("result", result)
    processLog.WithFields(fields).Logf("Request processing completed")
    return nil
}
```
