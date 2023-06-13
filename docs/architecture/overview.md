# Architecture

```mermaid
flowchart TD
    CommandExecutor{{"dogma.CommandExecutor"}}

    subgraph App["Dogma Application"]
        AggregateMessageHandler{{"dogma.AggregateMessageHandler"}}
        ProjectionMessageHandler{{"dogma.ProjectionMessageHandler"}}
        IntegrationMessageHandler{{"dogma.IntegrationMessageHandler"}}
        ProcessMessageHandler{{"dogma.ProcessMessageHandler"}}
    end

    subgraph Engine["Veracity Engine"]
        CommandRouter["Command\nRouter"]

        AggregateSubsystem["Aggregate\nSubsystem"]
        IntegrationSubsystem["Integration\nSubsystem"]
        ProcessSubsystem["Process\nSubsystem"]
        ProjectionSubsystem["Projection\nSubsystem"]

        EventStreamSubsystem["Event Stream\nSubsystem"]
    end

    CommandExecutor --> CommandRouter

    CommandRouter --> AggregateSubsystem
    CommandRouter --> IntegrationSubsystem

    AggregateSubsystem <--> AggregateMessageHandler
    AggregateSubsystem --> EventStreamSubsystem

    IntegrationSubsystem <--> IntegrationMessageHandler
    IntegrationMessageHandler -.- ThirdParty(("Third-party System"))
    IntegrationSubsystem --> EventStreamSubsystem

    EventStreamSubsystem --> ProcessSubsystem
    ProcessSubsystem <--> ProcessMessageHandler
    ProcessSubsystem --> CommandRouter

    EventStreamSubsystem --> ProjectionSubsystem
    ProjectionSubsystem --> ProjectionMessageHandler
    ProjectionMessageHandler -.- ReadModel[("Read Model")]

    click CommandExecutor href "https://pkg.go.dev/github.com/dogmatiq/dogma#CommandExecutor" "View Documentation" _blank
    click AggregateMessageHandler href "https://pkg.go.dev/github.com/dogmatiq/dogma#AggregateMessageHandler" "View Documentation" _blank
    click ProcessMessageHandler href "https://pkg.go.dev/github.com/dogmatiq/dogma#ProcessMessageHandler" "View Documentation" _blank
    click IntegrationMessageHandler href "https://pkg.go.dev/github.com/dogmatiq/dogma#IntegrationMessageHandler" "View Documentation" _blank
    click ProjectionMessageHandler href "https://pkg.go.dev/github.com/dogmatiq/dogma#ProjectionMessageHandler" "View Documentation" _blank

    click CommandRouter href "./command-router.md" "Drill Down" _parent
```
