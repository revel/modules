#FastHTTP
This module is the for [FastHTTP](https://github.com/valyala/fasthttp) server engine.
It does not support WebSockets.

###App.conf
- **server.engine** You must set this to `fasthttp` in order to use this server engine

###Other Notes
All features from supported by a regular HTTP engine is supported by this server engine.
Memory usage is decreased because this engine makes reuse of allocated structures to
handle requests. This should increase overall runtime performance and throughput. 
