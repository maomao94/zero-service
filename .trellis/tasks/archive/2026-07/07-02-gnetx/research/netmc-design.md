# Research: netmc framework design (for gnetx)

- **Query**: Reverse-engineer yezhihao's netmc (Java/Netty MVC framework) to design an analogous Go tool `gnetx` on top of gnet
- **Scope**: mixed (external netmc source @ `/Users/hehanpeng/IdeaProjects/netmc` + local consumer `allcore-sip` @ `/Users/hehanpeng/IdeaProjects/qinghai-biz/.../allcore-sip`)
- **Date**: 2026-07-02

> Source of truth: the netmc source checkout at `/Users/hehanpeng/IdeaProjects/netmc` (pom version **4.0.3**, Java 17). The consumer `allcore-sip` pins netmc **3.0.8** in its pom, but every API surface the consumer uses (`Session.notify/request/response` returning `Mono`, `HandlerInterceptor` 5-method contract, `MessageEncoder/Decoder.decode(ByteBuf, Session)`, `@Endpoint/@Mapping/@Async`) is byte-for-byte identical to the 4.0.3 source read here. The contracts are stable across these versions; the design below is faithful. See Caveats for the one minor delta.

---

## 1. Overall architecture (layers / packages)

netmc is a **Netty-based MVC framework for custom TCP/UDP binary protocols**. The README states the goal plainly: "基于Netty实现Mvc开发模式" (implement the traditional MVC development pattern on top of Netty). It maps Spring-MVC concepts onto a binary message bus:

| Spring MVC concept | netmc equivalent |
|---|---|
| `@Controller` | `@Endpoint` (class-level, on a "message access point") |
| `@RequestMapping` | `@Mapping(types={...}, desc="...")` (method-level, keyed by **int message id**) |
| `HandlerInterceptor` | `HandlerInterceptor<T extends Message>` (5-method contract) |
| `HandlerMapping` | `HandlerMapping.getHandler(int messageId)` |
| `View`/response | return value of the handler method (a `Message`) |
| `HttpRequest/Response` | `Message` (in) + `Session` (channel/state) |

### Package layout (source tree, `src/main/java/io/github/yezhihao/netmc/`)

```
netmc/
├── NettyConfig.java          # Builder + immutable config holder; constructs TCPServer or UDPServer
├── Server.java               # abstract base: start()/stop(), boss/worker/business pools
├── TCPServer.java            # NIO TCP ServerBootstrap + pipeline assembly
├── UDPServer.java            # NIO UDP DatagramChannel + pipeline assembly
├── codec/                    # USER-FACING codec contracts
│   ├── MessageEncoder.java   #    interface: ByteBuf encode(T, Session)
│   ├── MessageDecoder.java   #    interface: T decode(ByteBuf, Session)
│   ├── Delimiter.java        #    byte[] value + boolean strip
│   └── LengthField.java      # length-field framing params (offset/len/adjust/strip)
├── core/                     # dispatch + processing (the "MVC" core)
│   ├── model/Message.java    #    interface: getClientId()/getMessageId()/getSerialNo()
│   ├── model/Response.java   #    interface: getResponseSerialNo()  (for request/response correlation)
│   ├── HandlerMapping.java   #    interface: Handler getHandler(int messageId)
│   ├── HandlerInterceptor.java  # 5-method Around-style contract
│   ├── AbstractHandlerMapping.java  # reflects @Mapping methods into a HashMap<id,Handler>
│   ├── DefaultHandlerMapping.java   # scans a package for @Endpoint classes (plain-Java, no Spring)
│   ├── SpringHandlerMapping.java    # Spring variant: ApplicationContextAware, picks up @Endpoint beans
│   ├── annotation/{Endpoint,Mapping,Async,AsyncBatch}.java
│   └── handler/{Handler,SimpleHandler,AsyncBatchHandler}.java  # reflective invokers
├── handler/                  # NETTY-PIPELINE glue (framework-internal, user does NOT touch)
│   ├── TCPMessageAdapter.java        # channelActive/Inactive/idle/exception -> Session lifecycle
│   ├── UDPMessageAdapter.java        # UDP equivalent; owns its own idle-sweep thread
│   ├── MessageDecoderWrapper.java    # wraps user MessageDecoder as an inbound handler
│   ├── MessageEncoderWrapper.java    # wraps user MessageEncoder as an outbound handler
│   ├── DispatcherHandler.java        # the "front controller": lookup -> invoke -> interceptors
│   ├── DelimiterBasedFrameDecoder.java
│   ├── DynamicLengthFieldBasedFrameDecoder.java
│   └── LengthFieldAndDelimiterFrameDecoder.java
├── session/                  # connection state + request/response machinery
│   ├── Session.java          #    the per-connection context: attrs, serialNo, register, notify/request/response
│   ├── SessionManager.java   #    ConcurrentHashMap<sessionId,Session> + Caffeine offline cache + listener
│   ├── SessionListener.java  #    interface: sessionCreated/Registered/Destroyed (all default no-op)
│   └── Packet.java           #    envelope carrying (Session, Message, ByteBuf) through the pipeline
└── util/                     # Client (test raw-socket sender), ClassUtils, ByteBufUtils, IntTool, Stopwatch, Adapter{Collection,List,Map,Set}, VirtualList
```

### Three logical layers

1. **Transport layer** (`Server`/`TCPServer`/`UDPServer` + `handler/*Adapter*`): owns Netty `EventLoopGroup`s, assembles the `ChannelPipeline`, wires `IdleStateHandler`, frame decoder, adapter, decoder/encoder wrappers, dispatcher. **User never subclasses these** — they are configured entirely through `NettyConfig.Builder`.
2. **MVC/dispatch layer** (`core/*` + `handler/DispatcherHandler`): the "front controller". `DispatcherHandler.channelRead` → `HandlerMapping.getHandler(messageId)` → `Handler.invoke` → `HandlerInterceptor` around-advice. This is the layer that makes netmc feel like Spring MVC.
3. **Session/codec layer** (`session/*` + `codec/*`): the **user-facing extension points**. A user plugs in `MessageEncoder`, `MessageDecoder`, `HandlerMapping` (or `@Endpoint` classes), `HandlerInterceptor`, `SessionListener`, and `Message` model classes. Everything else is framework.

### Runtime dependencies (pom.xml)
- `io.netty:netty-handler` 4.1.119.Final
- `com.github.benmanes.caffeine:caffeine` 3.1.8 (offline message cache)
- `io.projectreactor:reactor-core` 3.6.15 (the `Mono`-based request/response API)
- `org.springframework:spring-context` 6.1.18 (`provided` — only for `SpringHandlerMapping`)
- `org.slf4j:slf4j-api` 2.0.17

The Reactor dependency is load-bearing: `Session.request(...)` returns `Mono<T>` and correlates the response via a `MonoSink` keyed on `responseClass.getName() + '.' + serialNo`. This is netmc's answer to async request/response over a single TCP connection.

---

## 2. Server-side API: booting a TCP server

Everything is built through one fluent builder. There is **no `TCPServer` constructor users call** — `NettyConfig.Builder.build()` picks `TCPServer` or `UDPServer` based on `enableUDP`.

### The builder (`NettyConfig.java`)

```java
public static class Builder {
    private int workerCore;                 // default: availableProcessors()+2
    private int businessCore;               // default: max(1, processors>>1)
    private int readerIdleTime = 240;       // seconds; IdleStateHandler reader idle
    private int writerIdleTime = 0;
    private int allIdleTime = 0;
    private Integer port;                   // REQUIRED
    private Integer maxFrameLength;         // REQUIRED for TCP
    private LengthField lengthField;        // pick ONE framing strategy:
    private Delimiter[] delimiters;         //   lengthField OR delimiters (TCP needs at least one)
    private MessageDecoder decoder;         // REQUIRED
    private MessageEncoder encoder;         // REQUIRED
    private HandlerMapping handlerMapping;  // REQUIRED
    private HandlerInterceptor handlerInterceptor; // REQUIRED
    private SessionManager sessionManager;  // optional; default new SessionManager()
    private boolean enableUDP;              // default false -> TCPServer
    private String name;                    // default "TCP"/"UDP", used for thread names + logs
    public Server build() { ... }
}
```

Validation (constructor, `NettyConfig.java:54-66`): `port`, `decoder`, `encoder`, `handlerMapping`, `handlerInterceptor` are non-null; for TCP, `maxFrameLength` must be positive and at least one of `delimiters`/`lengthField` must be set.

### TCP pipeline assembly (`TCPServer.java:41-57`)

For each accepted `NioSocketChannel`, the pipeline is (in order):

```java
.addLast(new IdleStateHandler(readerIdleTime, writerIdleTime, allIdleTime))
.addLast("frameDecoder", frameDecoder())   // LengthFieldBasedFrameDecoder | LengthFieldAndDelimiter | DelimiterBasedFrameDecoder
.addLast("adapter",   new TCPMessageAdapter(sessionManager))   // bytes -> Session -> Packet
.addLast("decoder",   new MessageDecoderWrapper(decoder))      // ByteBuf -> Message
.addLast("encoder",   new MessageEncoderWrapper(encoder))      // Message -> ByteBuf
.addLast("dispatcher",new DispatcherHandler(handlerMapping, handlerInterceptor, businessGroup))
```

Note: all of `adapter/decoder/encoder/dispatcher` are `@ChannelHandler.Sharable` singletons shared across all child channels — only `IdleStateHandler` and the frame decoder are per-channel.

### Thread model (`TCPServer.java:30-34`, `Server.java`)
- `bossGroup` = `NioEventLoopGroup(1, ...)` (accept)
- `workerGroup` = `NioEventLoopGroup(workerCore, ...)` (IO + the non-async handler path runs here)
- `businessGroup` = `ThreadPoolExecutor(businessCore, ...)` named `<name>-B` (only used when a handler is `@Async` — see §7)
- `start()` (`Server.java:30`): `bootstrap.bind(port).awaitUninterruptibly()`, registers a close-future listener to auto-`stop()`, sets `isRunning`. `synchronized`.
- `stop()`: graceful shutdown of boss/worker, `shutdown()` (not graceful) of businessGroup.

### Frame decoder selection (`TCPServer.frameDecoder()`, lines 61-72)
- if `lengthField != null` AND `delimiters != null` → `LengthFieldAndDelimiterFrameDecoder` (handles protocols where both a length field and delimiters can appear)
- else if `lengthField != null` → Netty's stock `LengthFieldBasedFrameDecoder`
- else → `DelimiterBasedFrameDecoder` (netmc's own, supports multiple delimiters + `strip` flag)

### Minimal boot example (from `QuickStart.java`, the test app)

```java
Server tcpServer = new NettyConfig.Builder()
        .setPort(7611)
        .setMaxFrameLength(2048)
        .setDelimiters(new byte[][]{"|".getBytes(UTF_8)})
        .setDecoder(new MyMessageDecoder())
        .setEncoder(new MyMessageEncoder())
        .setHandlerMapping(new DefaultHandlerMapping("io.github.yezhihao.netmc.endpoint"))
        .setHandlerInterceptor(new MyHandlerInterceptor())
        .setSessionManager(new SessionManager())
        .build();
tcpServer.start();
```

`DefaultHandlerMapping(packageName)` reflectively scans the package for `@Endpoint` classes and instantiates them via no-arg constructor (no DI). For Spring-managed beans, use `SpringHandlerMapping` instead (§5).

---

## 3. Client-side API

**netmc has no first-class TCP client for the request/response path.** The framework is server-centric: it accepts inbound connections, decodes, dispatches, and responds. Outbound "send a message to a connected device and await its reply" is done **server-side, through `Session`** (the server treats the connected device as a client it can push to).

### `util/Client.java` — test-only raw socket sender
This is a **test utility** (`src/test/...`), not a production client API. It is a `Closeable` with one method:

```java
public interface Client extends Closeable {
    void send(byte[] bytes) throws IOException;
    static Client[] TCP(String host, int port, int size) { ... }  // opens `size` java.net.Socket`s
    static Client[] UDP(String host, int port, int size) { ... }  // opens `size` DatagramSocket`s
}
```

It writes raw bytes — no codec, no `Message`, no `Session`. Used by `StressTest` to hammer the server.

### The real "client" path: server → device via `Session`
Production outbound messaging (the consumer's `MessageManager`) uses the **`Session` object of an already-connected, registered device**:

```java
// consumer: MessageManager.java — server pushes a request to a connected device, awaits reply
public <T> Mono<R<T>> requestR(String sessionId, TMessage request, Class<T> responseClass) {
    Session session = sessionManager.get(sessionId);       // lookup by device clientId
    if (session == null) return OFFLINE_RESULT;
    return session.request(request, responseClass)         // Mono-based request/response
            .map(message -> R.data(message))
            .timeout(Duration.ofMinutes(20), TIMEOUT_RESULT)
            .onErrorResume(e -> SENDFAIL_RESULT);
}
```

So the "client API" surface is really `Session.notify/request/response` (see §6). There is **no separate `Client` bootstrap class** in netmc — if a Go `gnetx` wants a symmetric client, this is a gap to fill (see §10).

---

## 4. Codec abstraction

### User-facing interfaces (`codec/`)

```java
// MessageEncoder.java
public interface MessageEncoder<T> {
    ByteBuf encode(T message, Session session);
}

// MessageDecoder.java
public interface MessageDecoder<T extends Message> {
    T decode(ByteBuf buf, Session session);
}
```

Two contracts, that's it. Both receive the **`Session`** as context (so encoding/decoding can consult per-connection state — the consumer uses this to inject `SessionHost` fields and to log with `channelId`). `decode` returns a `Message` (or subclass); `encode` returns a Netty `ByteBuf`.

### How they plug into the pipeline (`handler/MessageDecoderWrapper.java`, `MessageEncoderWrapper.java`)

The framework wraps the user interfaces in `@Sharable` Netty handlers — users never write Netty handlers directly.

```java
// MessageDecoderWrapper.channelRead  (inbound)
Packet packet = (Packet) msg;
ByteBuf input = packet.take();
Message message = decoder.decode(input, packet.session);   // <-- user code
if (message != null) ctx.fireChannelRead(packet.replace(message));
input.release();

// MessageEncoderWrapper.write  (outbound)
Packet packet = (Packet) msg;
ByteBuf output = packet.take();
if (output == null) output = encoder.encode(packet.message, packet.session);  // <-- user code
ctx.write(packet.wrap(output), promise);   // wrap() returns ByteBuf for TCP, DatagramPacket for UDP
```

Key detail: `Packet` flows through the whole pipeline, not raw `ByteBuf`/`Message`. `Packet.replace(message)` mutates the envelope in place so the downstream handler sees the decoded `Message` while keeping the same `Session` reference. `Packet.wrap(ByteBuf)` is the polymorphic outbound step — TCP returns the `ByteBuf` as-is, UDP wraps it in a `DatagramPacket` addressed to `session.remoteAddress()` (`Packet.java:46-68`).

### The consumer's adapter pattern (`TMessageAdapter.java`)
The consumer implements **both** `MessageEncoder<TMessage>` and `MessageDecoder<TMessage>` in one class, delegating to two plain collaborator classes:

```java
public class TMessageAdapter implements MessageEncoder<TMessage>, MessageDecoder<TMessage> {
    public ByteBuf encode(TMessage message, Session session) {
        ByteBuf output = messageEncoder.encode(message);     // pure framing, no Session
        encodeLog(session, message, output);                 // Session used only for logging
        return output;
    }
    public TMessage decode(ByteBuf input, Session session) {
        TMessage message = messageDecoder.decode(input);
        if (message != null) message.setSession(session);    // back-reference injected into the message
        decodeLog(session, message, input);
        return message;
    }
}
```

Note the consumer's `TMessageEncoder`/`TMessageDecoder` **do not take `Session`** (they predate the Session-aware overload); the adapter is the seam that adds logging + `message.setSession(session)`. A Go `gnetx` should bake the Session-aware signature in directly.

### Framing config objects (`Delimiter.java`, `LengthField.java`)
- `Delimiter(byte[] value, boolean strip)` — `strip=true` removes the delimiter from the frame (default), `strip=false` keeps it. The consumer uses `new Delimiter(new byte[]{(byte)0xEB,(byte)0x90}, false)` (the `EB90` end-marker is kept because the decoder validates it).
- `LengthField(prefix, maxFrameLength, lengthFieldOffset, lengthFieldLength, lengthAdjustment, initialBytesToStrip)` — wraps Netty's `LengthFieldBasedFrameDecoder` params + an optional magic prefix.

---

## 5. Protocol / Message abstraction

### The `Message` interface (`core/model/Message.java`)

```java
public interface Message extends Serializable {
    String getClientId();   // client identity -> used by Session.register + SessionManager.get
    int getMessageId();     // message type -> used to route to @Mapping(types=...)
    int getSerialNo();      // flow/sequence number -> used for request/response correlation
}
```

Three integers/strings, that's the entire contract a protocol message must satisfy. Everything else (XML body, fields, items) is the user's business.

### The `Response` marker interface (`core/model/Response.java`)

```java
public interface Response {
    int getResponseSerialNo();   // the serialNo of the REQUEST this message answers
}
```

A `Message` that **also** implements `Response` is treated specially by `Session.request/response` (§6): correlation is keyed by `responseClass.getName() + '.' + getResponseSerialNo()` instead of just the class name. This lets a single connection multiplex many in-flight requests.

### Routing: `@Endpoint` + `@Mapping` (annotations)

```java
@Target(TYPE)     @Retention(RUNTIME)
public @interface Endpoint {}

@Target(METHOD)   @Retention(RUNTIME)
public @interface Mapping {
    int[] types();            // one or more message ids this method handles
    String desc() default ""; // human-readable, used in logs/toString
}

@Target(METHOD)   @Retention(RUNTIME)
public @interface Async {}     // run handler on businessGroup, not IO thread

@Target(METHOD)   @Retention(RUNTIME)
public @interface AsyncBatch { // queue + batch; method must take a single List<Message>
    int poolSize() default 2;
    int maxElements() default 4000;
    int maxWait() default 1000;
}
```

### Handler method signature (reflective invocation, `core/handler/Handler.java:39-76`)
The handler method's parameters are **positionally introspected**, not named:
```java
Type[] types = targetMethod.getGenericParameterTypes();
// for each param: if assignable to Message -> MESSAGE; if assignable to Session -> SESSION
public <T extends Message> T invoke(T request, Session session) {
    Object[] args = new Object[parameterTypes.length];
    for (int i = 0; i < args.length; i++) {
        if (parameterTypes[i] == MESSAGE) args[i] = request;
        else if (parameterTypes[i] == SESSION) args[i] = session;
    }
    return (T) targetMethod.invoke(targetObject, args);
}
```
So a handler can be `void f(MyMsg, Session)`, `MyResp f(MyMsg, Session)`, `void f(MyMsg)`, `void f(Session)`, or even `void f(List<MyMsg>)` (for `@AsyncBatch`). `returnVoid` is detected from `getReturnType()`.

### Two `HandlerMapping` strategies
- **`DefaultHandlerMapping(String endpointPackage)`** (`core/DefaultHandlerMapping.java`): reflectively scans a package for `@Endpoint` classes, **instantiates each via `getDeclaredConstructor().newInstance()`** (no DI), then `registerHandlers(bean)` reflects `@Mapping` methods into a `HashMap<messageId, Handler>`. Used by the test app.
- **`SpringHandlerMapping`** (`core/SpringHandlerMapping.java`): implements `ApplicationContextAware`; on context init, `getBeansWithAnnotation(Endpoint.class)` and registers their `@Mapping` methods. **This is the path the consumer uses** — endpoints are Spring `@Component`s with `@Resource`-injected services.

`AbstractHandlerMapping.registerHandlers` (lines 26-53) is the shared reflection logic: for each `@Mapping` method, build either a `SimpleHandler` (sync, or `@Async`) or an `AsyncBatchHandler`, then `for (int type : mapping.types()) handlerMap.put(type, handler)` — one method can handle multiple message ids.

### How the consumer defines a protocol on top of netmc
1. **A constants interface** (`TSip.java`) — `int 注册指令 = 0xfb0001; ...` Magic ints encode `(type<<16)|command`. This is the "protocol vocabulary".
2. **Message classes** extending a base `TMessage implements Message`, annotated with **`@Message(...)` from a *separate* library `io.github.yezhihao:protostar`** (NOT netmc itself):
   ```java
   @Message(TSip.注册指令)                       // protostar annotation, not netmc
   @XStreamAlias("Root_T2511")                   // XStream XML binding
   public class T2511 extends TMessage {}
   ```
   `protostar`'s `@Message(value=...)` lets `TMessage.reflectMessageId()` read the id back off the class at runtime, and `MessageId.getClass(id)` builds a reverse `Map<Integer,Class>` by scanning `com.allcore.sip` for `@Message` classes (see `MessageId.java:34-43`).
3. **An `@Endpoint` class** (`SipEndpoint.java`) with `@Mapping(types = TSip.注册指令, ...)` methods that take the typed message subclass + `Session` and return a typed response.

> **Important design note**: netmc's own `@Mapping(types=...)` carries the int id at the method level. The consumer additionally uses `protostar`'s `@Message` at the **class** level to (a) let the decoder pick the right class from the wire id, and (b) reflect the id back. netmc and protostar are companion libraries by the same author but are decoupled — netmc only needs `Message.getMessageId()`; how the id is derived is the user's business.

---

## 6. Session / Channel abstraction

`Session` (`session/Session.java`) is the per-connection context object — the equivalent of an HTTP `HttpSession` crossed with a Netty `Channel` wrapper.

### Identity & state
```java
private String sessionId;          // set on register(); null until then
private String clientId;           // = Message.getClientId() at register time
private final Channel channel;     // underlying Netty channel
private final InetSocketAddress remoteAddress;
private final long creationTime;
private long lastAccessedTime;     // updated by access() on every inbound frame
private final Map<Object,Object> attributes;  // EnumMap if SessionManager given a key enum, else TreeMap
private final AtomicInteger serialNo = new AtomicInteger(0);  // 0..0xFFFF then wraps
private BiConsumer<Session,Message> requestInterceptor  = (s,m)->{};
private BiConsumer<Session,Message> responseInterceptor = (s,m)->{};
private final Map<String,MonoSink> topicSubscribers = new HashMap<>();  // pending request/response
```

### Registration & lifecycle
```java
public void register(Message message)            { register(message.getClientId(), message); }
public void register(String sessionId, Message m){ this.sessionId=sessionId; this.clientId=m.getClientId();
                                                   sessionManager.add(this); }  // fires sessionRegistered
public boolean isRegistered() { return sessionId != null; }
public void invalidate()      { if (isRegistered()) sessionManager.remove(this);  // fires sessionDestroyed
                                remover.apply(this); }                            // closes channel
public long access()          { lastAccessedTime = now; return lastAccessedTime; }
public int  nextSerialNo()    { return serialNo.getAndUpdate(p -> p>=0xFFFF?0:p+1); } // 16-bit wrap
```

The `remover` is a `Function<Session,Boolean>` injected at construction — for TCP it's `s -> { channel.close(); return true; }` (`SessionManager.newInstance` line 61-64); for UDP it removes the entry from `UDPMessageAdapter`'s `sessionMap` (line 61).

### Attribute storage (typed key pattern)
```java
public <T> T getAttribute(Object name)        { return (T) attributes.get(name); }
public void setAttribute(Object name, Object v){ attributes.put(name, v); }
```
The consumer uses an **enum `SessionKey`** as the key and a static helper to get a typed `SessionHost`:
```java
public enum SessionKey { SESSION_HOST;
    public static SessionHost getSessionHost(Session s) { return (SessionHost) s.getAttribute(SESSION_HOST); }
}
```
Passing the enum class to `new SessionManager(SessionKey.class, listener)` makes `Session.attributes` an `EnumMap` (line 62) — slightly more efficient. The consumer's `TBeanConfig` does exactly this.

### Outbound messaging — the `Mono`-based request/response (`Session.java:193-279`)
This is netmc's signature feature: **async request/response over a long-lived TCP connection, correlated by serial number**.

```java
// Fire-and-forget
public Mono<Void> notify(Message message) {
    requestInterceptor.accept(this, message);                // outbound hook (log/seq/attrs)
    Packet packet = Packet.of(this, message);
    return Mono.create(sink -> channel.writeAndFlush(packet).addListener(f -> {
        if (f.isSuccess()) sink.success(); else sink.error(f.cause());
    }));
}

// Request -> await a Response of class `responseClass`
public <T> Mono<T> request(Message request, Class<T> responseClass) {
    requestInterceptor.accept(this, request);
    String key = requestKey(request, responseClass);         // className + '.' + serialNo (if Response)
    Mono<T> receive = this.subscribe(key);                   // register a MonoSink; null if one already pending
    if (receive == null) return Rejected;                    // "客户端暂未响应，请勿重复发送"
    Packet packet = Packet.of(this, request);
    return Mono.create(sink -> channel.writeAndFlush(packet).addListener(...))
            .then(receive).doFinally(signal -> unsubscribe(key));
}

// Deliver an inbound message to a pending request
public boolean response(Message message) {
    responseInterceptor.accept(this, message);               // inbound hook
    MonoSink<Message> sink = topicSubscribers.get(responseKey(message));
    if (sink != null) { sink.success(message); return true; }
    return false;
}
```

Correlation keys (`requestKey`/`responseKey`, lines 263-279):
- If `responseClass` **is a `Response`**: key = `className + '.' + request.getSerialNo()` / matched by `((Response)msg).getResponseSerialNo()`. → **serial-number multiplexing**, many in-flight requests of the same type.
- If `responseClass` is **not** a `Response`: key = `className` only. → at most one pending request per class; the second concurrent `request` of the same non-Response class is rejected with `Mono.error(RejectedExecutionException)`.

The consumer's `SipEndpoint.response(TMessage, Session)` handler calls `session.response(message)` to complete the pending `Mono` when a `通用应答` arrives — see §9.

### `SessionManager` (`session/SessionManager.java`)
```java
private final ConcurrentHashMap<String,Session> sessionMap;     // by sessionId
private final Cache<String,Object> offlineCache;                // Caffeine, 10min TTL
private final SessionListener sessionListener;
private final Class<? extends Enum> sessionKeyClass;            // for EnumMap attributes
public Session get(String sessionId) { return sessionMap.get(sessionId); }
public Session newInstance(Channel channel) { ... fires sessionCreated ... }
protected void add(Session s)    { sessionMap.put(...); fires sessionRegistered; }
protected void remove(Session s) { sessionMap.remove(...); fires sessionDestroyed; }
public Object getOfflineCache(String clientId) / setOfflineCache(...)
```
`offlineCache` is a 10-minute TTL store so a message destined for a device that just dropped can be held — but note netmc itself doesn't auto-redeliver; the consumer must check it.

### `Packet` (`session/Packet.java`) — the pipeline envelope
```java
public abstract class Packet {
    public final Session session;
    public Message message;     // set after decode
    public ByteBuf byteBuf;     // set before encode / after frame
    public static Packet of(Session, Message);   // TCP or UDP variant
    public Packet replace(Message m) { this.message = m; return this; }
    public ByteBuf take() { ByteBuf t = byteBuf; byteBuf = null; return t; }  // null-safe, one-shot
    public abstract Object wrap(ByteBuf byteBuf);   // TCP: returns ByteBuf; UDP: returns DatagramPacket
}
```
`take()` is the one-shot handoff of the `ByteBuf` between frame-decoder → decoder-wrapper (so the wrapper can release it).

---

## 7. Handler / Interceptor pipeline

### Inbound dispatch — `DispatcherHandler.channelRead` (`handler/DispatcherHandler.java:46-97`)

This is the front controller. It is `@Sharable` and shared across all channels.

```java
public void channelRead(ChannelHandlerContext ctx, Object msg) {
    Packet packet = (Packet) msg;
    Message request = packet.message;
    Handler handler = handlerMapping.getHandler(request.getMessageId());

    if (handler == null) {                                   // no @Mapping for this id
        Message response = interceptor.notSupported(request, packet.session);
        if (response != null) ctx.writeAndFlush(packet.replace(response));
    } else {
        if (handler.async)                                   // @Async -> business thread pool
            executor.execute(() -> channelRead0(ctx, packet, handler));
        else
            channelRead0(ctx, packet, handler);              // sync -> IO thread
    }
}

private void channelRead0(ctx, packet, handler) {
    try {
        if (!interceptor.beforeHandle(request, session)) return;     // gate
        response = handler.invoke(request, session);                // reflective call
        if (handler.returnVoid)
            response = interceptor.successful(request, session);    // synthesize a response for void methods
        else
            interceptor.afterHandle(request, response, session);    // post-process a returned response
    } catch (InvocationTargetException e) {
        response = interceptor.exceptional(request, session, e.getTargetException());
    } catch (Exception e) {
        response = interceptor.exceptional(request, session, e);
    }
    if (time > 100) log.info("慢处理耗时{}ms", time);               // slow-handler warning
    if (response != null) ctx.writeAndFlush(packet.replace(response));  // send reply
}
```

### The 5-method `HandlerInterceptor<T>` contract (`core/HandlerInterceptor.java`)
```java
public interface HandlerInterceptor<T extends Message> {
    T notSupported(T request, Session session);                       // no handler found -> synthesize error reply
    boolean beforeHandle(T request, Session session);                 // gate; false == abort
    T successful(T request, Session session);                         // handler returned void -> synthesize reply
    void afterHandle(T request, T response, Session session);        // handler returned a value -> post-process
    T exceptional(T request, Session session, Throwable e);           // handler threw -> synthesize error reply
}
```
This is an **Around-style interceptor** with three "synthesize a response" hooks (`notSupported`, `successful`, `exceptional`) and two "observe" hooks (`beforeHandle`, `afterHandle`). The consumer's `SipHandlerInterceptor` uses all five to build a canonical `TResponse` with code 200/500 and to stamp routing fields (send/receive code swap, serialNo, time, XML).

### `@Async` vs sync (`core/handler/SimpleHandler.java`, `Handler.java`)
`Handler.async` is set from `method.isAnnotationPresent(Async.class)`. If async, `DispatcherHandler` dispatches `channelRead0` onto `businessGroup` so the IO thread is freed. The consumer marks heartbeat (`T2512`) and long-running report handlers (`T2513`) as `@Async`.

### `@AsyncBatch` — high-throughput batching (`core/handler/AsyncBatchHandler.java`)
For high-volume messages (the README cites JT808 `0x0200` position reports), `@AsyncBatch` runs a **dedicated `FixedThreadPool(poolSize)`** per method with a `ConcurrentLinkedQueue`:
```java
public T invoke(T request, Session session) { queue.offer(request); return null; }   // never blocks dispatcher
// worker loop: drain up to maxElements, invoke targetMethod with a VirtualList<Message>,
// sleep maxWait ms if queue drained < maxElements. Warns if queue > maxElements*poolSize*50.
```
The method signature must be `void f(List<X extends Message>)` (validated in constructor, lines 42-46). `VirtualList` is a list view over a pre-sized `Message[]` to avoid allocation. **Note: `@AsyncBatch` handlers run on their own pool, NOT `businessGroup`** — `Handler.async` is false for them, so the dispatcher invokes synchronously (which just enqueues), and the batch thread does the real work.

---

## 8. Lifecycle hooks

### Connection lifecycle — `TCPMessageAdapter` (`handler/TCPMessageAdapter.java`)
This `@Sharable` inbound handler is the **bridge between Netty channel events and `Session`/`SessionListener`**:

| Netty event | netmc behavior |
|---|---|
| `channelActive` | logs `<<<<< Connected<remoteAddress>` only (Session is lazily created on first read) |
| `channelRead` (first & subsequent) | `getSession(ctx)`: if `channel.attr(KEY)` is null, `sessionManager.newInstance(channel)` (fires `sessionCreated`); then `session.access()`; fires `Packet.of(session, buf)` downstream |
| `channelInactive` | `session.invalidate()` → `sessionManager.remove` (fires `sessionDestroyed`) + `channel.close()`; logs `>>>>> Disconnected` |
| `exceptionCaught` | `IOException` → warn "终端断开连接"; else warn "消息处理异常" (does NOT close channel) |
| `userEventTriggered(IdleStateEvent)` | logs "终端心跳超时" + `ctx.close()` (which triggers channelInactive → invalidate) |

The `Session` is stored on the Netty channel via `AttributeKey<Session>` named `Session.class.getName()` — one Session per channel, looked up on every read.

### UDP lifecycle — `UDPMessageAdapter` (`handler/UDPMessageAdapter.java`)
UDP has no per-channel `channelInactive`, so the adapter maintains its own `Map<InetSocketAddress,Session>` and runs a **daemon idle-sweep thread** started in `channelActive` (lines 73-100):
```java
for (;;) {
    long now = now();
    for (Session s : sessionMap.values()) {
        long time = readerIdleTime - (now - s.getLastAccessedTime());
        if (time <= 0) { log.warn("心跳超时"); s.invalidate(); }   // removes from map + fires destroyed
        else nextDelay = min(time, nextDelay);
    }
    Thread.sleep(nextDelay);
}
```
A `DelimiterBasedFrameImpl` subclass splits one datagram into multiple frames (UDP can carry several messages per packet).

### Session-state lifecycle — `SessionListener` (`session/SessionListener.java`)
```java
public interface SessionListener {
    default void sessionCreated(Session s) {}     // TCP: on first read; UDP: on first packet
    default void sessionRegistered(Session s) {}  // when Session.register adds to SessionManager
    default void sessionDestroyed(Session s) {}   // on invalidate (channel close / idle timeout)
}
```
All three are default no-op, so users implement only what they need. `SessionManager` wraps every listener call in try/catch (lines 65-70, 87-92, 97-102) so a listener bug can't break the pipeline.

The consumer's `SessionListener` uses `sessionCreated` to install the **outbound interceptors** (`requestInterceptor`/`responseInterceptor`) — these are per-Session `BiConsumer<Session,Message>` hooks invoked by `Session.notify/request/response` to log and stamp outbound messages (§9).

### Exception handling summary
- **Decode errors**: `MessageDecoderWrapper` catches, logs hex dump, rethrows as `DecoderException` (does NOT auto-close; propagates to `TCPMessageAdapter.exceptionCaught`).
- **Encode errors**: `MessageEncoderWrapper` catches, logs, rethrows as `EncoderException`.
- **Handler errors**: `DispatcherHandler.channelRead0` catches `InvocationTargetException` (unwraps target) + generic `Exception`, routes to `interceptor.exceptional(...)`; the returned `Message` (if any) is sent as the reply. The channel stays open.
- **IO errors** (broken pipe etc.): `TCPMessageAdapter.exceptionCaught` logs warn; the subsequent `channelInactive` cleans up the Session.

---

## 9. How the allcore-sip consumer uses netmc (concrete mapping)

The consumer is a Spring Boot app (`allcore-sip`) that implements a power-grid inspection protocol ("SIP", not SIP/RTP) on top of netmc. Every netmc extension point is mapped to a concrete consumer class:

### Wiring (Spring `@Configuration`)
**`TBeanConfig.java`** — declares the beans netmc needs:
```java
@Bean public HandlerMapping handlerMapping()              { return new SpringHandlerMapping(); }   // picks up @Endpoint @Components
@Bean public SipHandlerInterceptor handlerInterceptor()   { return new SipHandlerInterceptor(); }
@Bean public SessionListener sessionListener()            { return new SessionListener(); }
@Bean public SessionManager sessionManager(SessionListener l) { return new SessionManager(SessionKey.class, l); }  // EnumMap attrs
@Bean public TMessageAdapter messageAdapter() {
    return new WebLogAdapter(new TMessageEncoder(), new TMessageDecoder());   // WebLogAdapter extends TMessageAdapter, adds web logging
}
```
**`TConfig.java`** — builds the server via `NettyConfig.custom()`:
```java
return NettyConfig.custom()
        .setIdleStateTime(120, 0, 0)                       // 120s reader idle -> close
        .setPort(port)
        .setThreadGroup(workerCore, businessCore)
        .setMaxFrameLength(Integer.MAX_VALUE)
        .setDelimiters(new Delimiter(new byte[]{(byte)0xEB,(byte)0x90}, false))   // EB90 marker, NOT stripped
        .setDecoder(messageAdapter)                        // same object is both encoder & decoder
        .setEncoder(messageAdapter)
        .setHandlerMapping(handlerMapping)
        .setHandlerInterceptor(handlerInterceptor)
        .setSessionManager(sessionManager)
        .setName("SIP-TCP")
        .build();
```
**`SipTcpServerLifecycle.java`** — wraps the `Server` bean in Spring `SmartLifecycle` (phase `Integer.MAX_VALUE`, so it starts last) to call `server.start()`/`stop()` on context refresh. (The `@Bean(initMethod="start")` line is commented out in favor of the explicit lifecycle.)

### File → netmc extension point

| Consumer file | netmc class/contract it implements | What it does |
|---|---|---|
| `transport/basics/TMessage.java` | `implements io.github.yezhihao.netmc.core.model.Message` | Base message: serialNo, send/receiveCode, type/command, XML body, `toXML()` via XStream, `reflectMessageId()` via protostar `@Message` |
| `transport/protocol/server/T2511.java`, `TServer.java`, `T2512.java`, `TResponse.java` | extend `TMessage`; `TResponse implements Response` | Concrete messages. `@Message(TSip.xxx)` (protostar) binds class↔id; `@XStreamAlias` binds XML root |
| `transport/protocol/server/TResponse.java` | `implements io.github.yezhihao.netmc.core.model.Response` | `getResponseSerialNo() = receiveSerialNo` — enables serialNo-multiplexed request/response |
| `transport/codec/TMessageAdapter.java` (and `WebLogAdapter` subclass) | `implements MessageEncoder<TMessage>, MessageDecoder<TMessage>` | Delegates to `TMessageEncoder`/`Decoder`, adds Session-aware logging + `message.setSession(session)` |
| `transport/codec/TMessageEncoder.java` | plain collaborator (not a netmc interface) | Writes `EB90 + sendSerialNo(LE) + recvSerialNo(LE) + sessionSource + xmlLen(LE) + xml + EB90` |
| `transport/codec/TMessageDecoder.java` | plain collaborator | Reads the frame, parses XML root, derives `(type,command)→messageId` via `TUtils`, looks up class via `MessageId.getClass(id)`, XStream-deserializes |
| `transport/endpoint/SipEndpoint.java` | `@Endpoint @Component`; methods `@Mapping(types=TSip.xxx)` | The "controller": register/heartbeat/response/T2513 handlers. Returns `T2511_2514`/`TResponse`/`void`(`@Async`) |
| `transport/endpoint/SipHandlerInterceptor.java` | `implements HandlerInterceptor<TMessage>` | All 5 hooks: builds canonical `TResponse` (200/500), swaps send/receive codes, stamps serialNo + XML |
| `transport/endpoint/SessionListener.java` | `implements io.github.yezhihao.netmc.session.SessionListener` | `sessionCreated` installs `requestInterceptor`/`responseInterceptor` `BiConsumer`s (log + stamp outbound); logs created/registered/destroyed |
| `transport/endpoint/MessageManager.java` | uses `SessionManager` + `Session` | Server→device outbound API: `notifyR`/`requestR` returning `Mono<R<T>>` with 20-min timeout + offline/sendfail fallbacks |
| `transport/model/entity/SessionHost.java` | plain POJO | Per-session business state (device code, root node name, host code, internal channelId) |
| `transport/model/enums/SessionKey.java` | enum used as `Session` attribute key | `SESSION_HOST`; `getSessionHost(session)` typed accessor |
| `transport/commons/TSip.java` | plain `interface` of int constants | Protocol vocabulary; `(type<<16)|command` encode/decode helpers |
| `transport/commons/MessageId.java` | static registry | Scans `com.allcore.sip` for protostar `@Message` classes → `Map<Integer,Class>` for the decoder; `Map<Integer,String>` id→name for logs |

### Two-layer interceptor design the consumer relies on
The consumer uses **both** interceptor layers netmc exposes:

1. **`HandlerInterceptor`** (per-dispatch, on the IO/business thread): `beforeHandle`→`invoke`→`afterHandle`/`successful`/`exceptional`. This is where the canonical ACK (`TResponse` code 200/500) is synthesized and routing fields are stamped.
2. **`Session.requestInterceptor`/`responseInterceptor`** (per-outbound-message `BiConsumer`, installed in `sessionCreated`): this is where **outbound** messages get their serialNo assigned (`session.nextSerialNo()`), send/receive codes filled from `SessionHost`, XML rendered, and a DB log row written. It runs inside `Session.notify`/`request`/`response` — i.e. on whatever thread calls those (the dispatcher for replies, the business thread for `MessageManager` pushes).

This split is deliberate: `HandlerInterceptor` handles the *inbound request → reply* path; the `BiConsumer` interceptors handle *any outbound message* (replies AND server-initiated pushes), so logging/stamping is centralized regardless of origin.

### The request/response round-trip in the consumer
1. Device connects. First `EB90`-framed `T2511` (register) arrives → `TMessageDecoder` builds a `T2511` → `DispatcherHandler` → `SipEndpoint.T2511_2514(T2511, Session)` → `session.register(message)` (adds to `SessionManager`, fires `sessionRegistered`) → `setAttribute(SESSION_HOST, new SessionHost(...))` → returns `T2511_2514` (code 200 + heartbeat intervals).
2. Server pushes a command: `MessageManager.requestR(clientId, req, TResponse.class)` → `session.request(req, TResponse.class)` → `requestInterceptor` stamps serialNo/codes/XML → writes `Packet` → returns `Mono<TResponse>` that will be completed when a `TResponse` arrives.
3. Device sends a `通用应答` (`TResponse`) → decoded → `SipEndpoint.response(TMessage, Session)` → `session.response(message)` → looks up `topicSubscribers[TResponse.className + '.' + message.getResponseSerialNo()]` → `sink.success(message)` → the pending `Mono` emits → `MessageManager.requestR`'s `.map(R::data)` runs → business code gets the reply.
4. Heartbeat: `T2512` → `@Async @Mapping` handler returns void → `interceptor.successful` synthesizes nothing meaningful (the consumer's `successful` builds a `TResponse` but since `@Async` + void... actually `@Async` only changes the *thread*, not the reply path — `successful` still runs and its return is flushed). The consumer's heartbeat handler is `@Async void` and `successful` builds a generic ack.

---

## 10. Key design decisions for gnetx

### Worth copying

1. **The 3-method `Message` contract** (`getClientId`/`getMessageId`/`getSerialNo`) is the minimal, protocol-agnostic routing surface. A Go `gnetx` should define an equivalent `Message` interface (`ClientID() string` / `MessageID() int` / `SerialNo() int`) — everything else stays user-defined.

2. **`Response` marker + serialNo-keyed `Mono`/future correlation** is the single most valuable feature. Without it, "send a command to a device and await its ACK" requires the user to hand-roll a map. In Go, the equivalent is a `map[string]chan Message` or `map[string]func(Message)` keyed by `responseClass + serialNo`, with a `Request(ctx, msg, respType) (Message, error)` API returning a future/channel. **Keep the two correlation modes**: serialNo-multiplexed (when the response implements `Response`) and singleton-per-class (when it doesn't).

3. **Routing by int `messageId` via struct-tag/annotation** is dramatically simpler than type-switch dispatch. In Go, mirror this with either struct tags (`type T2511 struct { ... }` registered against an int id via `gnetx.Register(id, T2511{}, handler)`) or a method registry. The `HandlerMapping.getHandler(int)` → `Handler.invoke` reflective call maps cleanly to Go's `reflect` or (faster) a registered `func(msg, session)`.

4. **Two-layer interceptor split** (per-dispatch `HandlerInterceptor` + per-outbound-message `BiConsumer` on Session) cleanly separates "inbound request handling" from "outbound message stamping/logging". Replicate this: a `HandlerInterceptor` interface with `Before/After/Success/Exception/NotSupported`, plus per-Session `OnSend/OnReceive` hooks. This lets logging/serialNo-assignment happen in exactly one place for both replies and server-initiated pushes.

5. **`Session` as the unified per-connection context** (attributes map + serialNo counter + register/invalidate + outbound API). Don't expose the raw `net.Conn` to user handlers; wrap it. The `setAttribute`/`getAttribute` map with a typed-key helper enum is a clean, allocation-light pattern (Go: `interface{}` keys with typed getter funcs).

6. **Config-driven pipeline assembly** (a `Config` builder that takes codec/interceptor/mapping/sessionmanager and hides the gnet pipeline). Users should never touch gnet's `EventHandler.OnTraffic` directly. The builder should validate required fields (port, decoder, encoder, mapping, interceptor) like `NettyConfig` does.

7. **`SessionListener` default-no-op interface** for created/registered/destroyed — easy to adopt selectively. In Go, an interface with embedded `noopSessionListener` struct so users embed it and override only what they need.

8. **Sharable singletons for codec/adapter/dispatcher** — instantiate once, share across all connections. In gnet this is natural (the `EventHandler` is already shared); keep the codec/interceptor/dispatcher as single shared objects, not per-connection.

9. **Slow-handler warning** (`if time > 100ms log`) and the **`@Async` escape hatch** to a business goroutine pool — directly portable. Go's equivalent: an optional `async bool` on handler registration that dispatches via a worker pool instead of inline on the reactor goroutine. **The `@AsyncBatch` queue+drain pattern** is also worth porting for high-volume telemetry messages (Go: a buffered channel + a flush goroutine with `maxElements`/`maxWait`).

10. **`Packet`-style envelope through the pipeline** carrying `(Session, Message, []byte)` so decode/encode/dispatch all share the same context without re-fetching. In Go this is a `struct { S *Session; Msg Message; Buf []byte }` passed through a small in-process pipeline.

### Worth avoiding / improving

1. **No first-class client.** netmc is server-only; outbound messaging reuses the server-side `Session` of a connected device. If `gnetx` wants to *connect out* to devices (active client), it needs a separate `Client`/`Connector` bootstrap that produces `Session` objects with the same `notify/request/response` API. netmc's `util/Client` is a test stub and not a model.

2. **Reflective handler dispatch + `InvocationTargetException` unwrapping** is verbose. In Go, prefer **pre-registered typed handler funcs** (`gnetx.Handle(id, func(s *Session, m *T2511) (*TResponse, error))`) over runtime reflection — type-safe, faster, and the compiler catches signature drift. Keep the option of a `HandleAny(func(*Session, Message))` for fallback.

3. **The `@Message`/`protostar` coupling is implicit.** netmc's `Message.getMessageId()` returns an int but netmc itself has no way to map `id → message class` for decoding — the consumer had to build its own `MessageId` registry by scanning for an external library's (`protostar`) annotation. `gnetx` should ship a **first-class `MessageRegistry`**: `Register(id int, factory func() Message)` so the decoder can instantiate the right concrete type without the user writing a scanner. This is a real papercut in the consumer (see `MessageId.java` — 74 lines of reflection that should have been framework).

4. **`Mono`/Reactor dependency for request/response.** It's powerful but heavyweight and unfamiliar to Go devs. The Go equivalent is trivially `chan Message` + `context.Context` for timeout/cancel — no external dependency, idiomatic, and composes with `select`. Do NOT pull in a Reactor port; use channels.

5. **Singleton-per-class correlation for non-`Response` messages** (only one pending request of a given class) is surprising — a second concurrent `request` for the same non-Response class returns `Mono.error(RejectedExecutionException)`. Document this clearly or, better, **always key by serialNo** and drop the non-Response branch (it's a footgun).

6. **`AsyncBatchHandler`'s busy-loop with `Thread.sleep(maxWait)`** and a `VirtualList` over a pre-sized array is a Java-specific optimization. In Go, a buffered channel + a `select` with `time.After(maxWait)` is idiomatic and avoids the manual array management. The *semantics* (batch up to N or until T) are worth keeping; the implementation is not.

7. **`setMaxFrameLength(Integer.MAX_VALUE)`** in the consumer disables frame-length protection entirely — the protocol relies solely on the `EB90` delimiter. `gnetx` should encourage length-field framing and make `MaxFrameLength` a hard requirement (netmc only requires it for TCP, allows `Integer` nullability; Go should make it a required `int`).

8. **`TCPMessageAdapter.exceptionCaught` does not close the channel on non-IO exceptions** — a poisoned connection can linger. Consider an explicit "close on unrecoverable decode/encode exception" policy, or at least a configurable one.

9. **`SessionManager.newInstance` for TCP is called on first read, not `channelActive`** — so `sessionCreated` fires lazily. This is fine but surprising; document that `sessionCreated` ≠ "TCP connected". For UDP, `sessionCreated` fires on the first packet from a given sender.

10. **`offlineCache` (Caffeine 10-min TTL) is set/get only** — netmc never auto-retries or flushes pending messages to a reconnecting device. The consumer doesn't use it either. `gnetx` could either drop this or make it a real "reconnect redelivery queue" with a callback; in its current half-built form it's dead weight.

---

## Caveats / Not Found

- **Version discrepancy**: netmc source checkout is pom **4.0.3** (Java 17, Netty 4.1.119, Reactor 3.6.15); consumer pins **3.0.8**. The API surface the consumer uses is identical to 4.0.3 as read — `Session.notify/request/response` Mono API, 5-method `HandlerInterceptor`, `MessageEncoder/Decoder.decode(ByteBuf, Session)` all match. The 3.0.8→4.0.3 delta (if any) is not in the paths the consumer exercises. **If precise 3.0.8 behavior matters, diff the netmc git history between the two tags** — not done here.
- **`protostar` (`io.github.yezhihao:protostar` 3.0.8)** is a *separate* companion library that provides `@Message(int[])` class-level annotation + `ClassUtils.getClassList(package, annotation)` scanner. The consumer depends on it for id→class mapping, but **netmc itself does not depend on protostar** (not in netmc's pom). This coupling is a consumer convention, not a framework feature. protostar source was not read for this research; only its annotation surface as observed from the consumer.
- **`StressTest.java`** was listed in the source tree but not read — it exercises the `util.Client` raw-socket sender for load testing. Not relevant to the design contract.
- **`DynamicLengthFieldBasedFrameDecoder.java`** and **`LengthFieldAndDelimiterFrameDecoder.java`** were not read in full; their existence confirms netmc supports mixed length+delimiter framing, but the consumer uses pure delimiter framing (`EB90`), so the length-field variants' exact semantics were not traced.
- **No first-class client bootstrap exists in netmc.** Confirmed by reading `util/Client.java` (a test utility in `src/test/`) and the absence of any `Client.java`/`TCPClient.java` in `src/main/`. Outbound messaging is server-side-via-`Session` only.
- **UDP path is documented from source but the consumer is TCP-only** (`TConfig` builds a TCP server, no `setEnableUDP(true)`). The UDP `IdleState` sweep-thread behavior and `DelimiterBasedFrameImpl` one-datagram-many-frames split are described from source, not from consumer usage.
