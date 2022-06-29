#

## Server 

- 服务节点几大种
    1. 同一节点名，多台服务，（可以配置唯一编号，或配置唯一静态地址: Host:Port)
    2. 一个节点命名只存在一台


## server name 

```
|           name           |
| nodename | sname | fname |
| nodename |   apiname     |
```

    Node.Struct.Function
    - nodename: Node
    - sname: Struct         // server struct name
    - fname: Function       // struct function name
    - apiname: Struct.Function



## Design Background

- All node server is same to connection
    + Discovery node
    + Gate node
    + Server node
    + Logs node

- Built-in receiving interface
    + receive function mapping
    + receive server node list update
    + receive node status to balance


## Function Mapping 
    - [1-99], 内部接口序号
    - [100-65535], 外部接口序号
 