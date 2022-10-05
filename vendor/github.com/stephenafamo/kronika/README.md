# Kronika - The keeper of time

Kronika adds some extra utility around the standard time package. It does not have any other external dependencies.

## WaitUntil

```go
kronika.WaitUntil(ctx context.Context, t time.Time)
```

WaitUntil will block **until** the given time. Unlike using `time.After` or `time.Timer`, it can be cancelled by cancelling the context.

## WaitFor

```go
kronika.WaitFor(ctx context.Context, diff time.Duration)
```

WaitFor will block for the specified duration. Unlike using `time.After` or `time.Timer`, it can be cancelled by cancelling the context.

## Every

```go
kronika.Every(ctx context.Context, start time.Time, interval time.Duration) <-chan time.Time
```

`kronika.Every()` works like a cron job. It is supplied a start time, and an interval, and it will send the current time to the returned channel at the specified intervals. Again, it can always be cancelled using the context.

Here are some examples of using `Every`

### Run on Tuesdays by 2pm

```go
ctx := context.Background()

start, err := time.Parse(
    "2006-01-02 15:04:05",
    "2019-09-17 14:00:00",
) // is a tuesday
if err != nil {
    panic(err)
}

interval := time.Hour * 24 * 7 // 1 week

for t := range kronika.Every(ctx, start, interval) {
    // Perform action here
    log.Println(t.Format("2006-01-02 15:04:05"))
}
```

### Run every hour, on the hour

```go
ctx := context.Background()

start, err := time.Parse(
    "2006-01-02 15:04:05",
    "2019-09-17 14:00:00",
) // any time in the past works but it should be on the hour
if err != nil {
    panic(err)
}

interval := time.Hour // 1 hour

for t := range kronika.Every(ctx, start, interval) {
    // Perform action here
    log.Println(t.Format("2006-01-02 15:04:05"))
}
```

### Run every 10 minutes, starting in a week

```go
ctx := context.Background()

// see https://golang.org/pkg/time/#Time.AddDate
start, err := time.Now().AddDate(0, 0, 7) 
if err != nil {
    panic(err)
}

interval := time.Minute * 10 // 10 minutes

for t := range kronika.Every(ctx, start, interval) {
    // Perform action here
    log.Println(t.Format("2006-01-02 15:04:05"))
}
```