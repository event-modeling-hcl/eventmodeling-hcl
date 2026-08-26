slice "example" {
  title      = "Example"
  slice_type = "STATE_CHANGE"

  command "command" {
    title = "Command"
    type  = "COMMAND"

    field "name" {
      type = "Text"
    }
  }
}
