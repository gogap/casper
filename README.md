casper
======

A FBP framwoker 

	一个组件，可以实现一个业务功能, 也可以实现多个。
	一条消息，可以流经任意多个组件。
	组件加工消息并传递， 同时跟据消息触发业务逻辑。

	每个组件，可以配置成一个一般业务组件， 也可以配置成为一个服务。
	服务本身也是一个组件。
	服务相对一般组件所增加的功能， 就是：可以接收外部请求。

	服务可以向外提供很多不同的具体"服务".
	每个具体服务对应一条具体的消息格式， 和， 一串业务组件调用链. "graph"定义此链。
	
	现在实现的zmq的消息方式， HTTP和zmq的服务.



