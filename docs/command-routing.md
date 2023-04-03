```mermaid
flowchart TD
    subgraph Application
        Begin("User calls ExecuteCommand()")
    end

    subgraph Dogma Engine
        Begin --> Execute

        Execute[Execute the command]
        Valid?{Is the command valid?}
        RouteToHandler[Route the command\nto a handler]
        HandlerType?{Which handler type\nhandles this command?}
        RouteToInstance["Call RouteCommandToInstance()"]
        ThisNode?{"Does this node currently\nmanage the target instance?"}
        Handle["Call HandleCommand()"]
        Reachable?{"Is the appropriate\nnode reachable?"}
        Send[Send command to\nthe other node]

        Execute --> Valid?

        Valid? -- YES --> RouteToHandler
        Valid? -- NO --> NotValid(Return an error)

        RouteToHandler --> HandlerType?

        HandlerType? -- AGGREGATE --> RouteToInstance
        HandlerType? -- INTEGRATION --> Handle
        HandlerType? -- UNKNOWN --> Unknown(Return an error)

        RouteToInstance --> ThisNode?

        ThisNode? -- YES --> Handle
        ThisNode? -- NO --> Reachable?

        Reachable? -- YES --> Send
        Reachable? -- NO --> Handle
    end

    API{{Inter-node gRPC API}}
    Send -.-> API
    API -.-> Execute

    Done(Done)
    Handle --> Done

```
