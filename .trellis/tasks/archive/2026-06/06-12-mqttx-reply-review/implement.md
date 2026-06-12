# Implementation Plan: mqttx reply review

## Checklist

- Update `ReplyMessage[T]` from `ID` to `Tid` and rename internal parameters to `tid` where they represent the unique reply ID.
- Change `ReplyDecoder[T]` from function type to Go-style interface with `Decode(...)`.
- Add `ReplyDecoderFunc[T]` adapter and tests for function decoder usage.
- Replace public `WithReplyHandler(topic string, h ConsumeHandler)` with `WithReplyRouter[T](topic string, router *ReplyRouter[T]) Option`.
- Keep internal reply storage as `ConsumeHandler` only after API boundary has enforced `*ReplyRouter[T]`.
- Keep dispatcher lookup keyed by callback `topicTemplate`; do not add a second wildcard matching layer.
- Update dispatcher tests to assert precise template routing and no-handler behavior.
- Ensure dispatch order is reply routers first, then regular handlers, while suppressing `ErrReplyNotMatched` logs.
- Preserve same-topic regular handler execution order through handler slices.
- Keep `getAllTopicTemplates()` map-based; test only inclusion and de-duplication.
- Add full reply router and dispatcher tests in `common/mqttx`.
- Run `gofmt` on touched Go files.
- Run `go test -count=1 ./common/mqttx/`.

## Test Matrix

- `ReplyRouter.HandleReply` resolves a pending `tid`.
- `ReplyRouter.Consume` returns `ErrReplyNotMatched` for decoded but unmatched `tid`.
- Nil decoder returns `ErrNilDecoder`.
- Decoder error is propagated.
- Empty `Tid` returns the empty ID/tid sentinel error.
- `Do` sends after registering and returns resolved value.
- `Do` cleans pending state when send fails.
- `Reject` rejects pending request.
- `Close` rejects pending request.
- `WithReplyRouter` registers reply topic for restore subscription.
- Dispatcher runs reply router before regular handler and still runs regular handler.
- Dispatcher uses callback `topicTemplate` as the exact lookup key.
- Same-topic regular handlers run in registration order.
- `getAllTopicTemplates()` includes normal and reply topic templates once.

## Risk Points

- Renaming `ReplyMessage.ID` to `Tid` is a breaking change for any current local callers; search before edit and update all references.
- MQTT wildcard matching belongs to the MQTT client subscription layer, not dispatcher.
- Reply decode errors should be visible, but decoded-unmatched replies should not create noisy error logs.
- Topic map order must not be asserted in tests.

## Validation Commands

- `gofmt -w common/mqttx/*.go`
- `go test -count=1 ./common/mqttx/`
