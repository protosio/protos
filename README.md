# Protos

- declarative core
- imperative shell - this should implement what the declarative layer says. But these actions should be in some cases accessible so I can do some actions read-nly from the cli or another client
- deletion shoudl reqiure some kind of confirmation
- there should be a method to say that the imperative layer matches the declarative layer, some kind of OK check, which could be used by the CLI or other clients. Not sure the shape yet
- when stopping protosd, the networking and local VMs should stay active if we don't close the hostagent
