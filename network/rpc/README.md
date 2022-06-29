# 

## 节点注册

    节点请求注册后，发现节点将保留节点信息，并根据注册的节点信息进行测试端口，保留连接并记录下来给其他节点获取



## Encode 
```
Request body make and parse:
	Head 8 bytes: (max all 14 bytes)
		2 byte: body[0:2]: check body start [1,0]
		3 byte: body[2:5]: request no
		1 byte: body[5]: BodyReqDataNil
		2 byte: body[6:8]: interface, server api, max: 65535

	BodyReqDataNil:
		2 byte: body[len(body)-2:]: last number to be end [0,1]

	BodyWholeData:
		2 byte: body[8:10]: bucker number
		2 byte: body[10:12]: body lenght
		2 byte: body[len(body)-2:]: last number to be end [0,1]

	BodyReqStart:
		2 byte: body[8:10]: bucker number
		2 byte: body[10:12]: sort number
		2 byte: body[len(body)-2:]: last number to be end [0,1]

	BodyReqMiddle
		2 byte: body[8:10]: bucker number
		2 byte: body[10:12]: sort number
		2 byte: body[len(body)-2:]: last number to be end [0,1]

	BodyReqFinaly:
		2 byte: body[8:10]: bucker number
		2 byte: body[10:12]: body lenght
		2 byte: body[len(body)-2:]: last number to be end [0,1]
```