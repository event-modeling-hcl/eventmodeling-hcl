# The command's output names no context-owned event.
bounded_context "example" {
  title = "Example"
}

state_change "example" {
  title = "Example"

  command "command" {
    title = "Command"
    to    = [event.missing.nonexistent_event]
  }
}
