slice "example" {
  title      = "Example"
  slice_type = "STATE_CHANGE"

  command "command" {
    title = "Command"
    type  = "COMMAND"
  }

  event "event" {
    title = "Event"
    type  = "EVENT"
  }

  readmodel "readmodel" {
    title = "Read model"
    type  = "READMODEL"
  }

  screen "screen" {
    title = "Screen"
    type  = "SCREEN"
  }

  processor "processor" {
    title = "Processor"
    type  = "AUTOMATION"
  }

  screen_image "image" {
    title = "Image"
  }

  table "table" {
    title = "Table"
  }

  specification "specification" {
    title     = "Specification"
    linked_id = "example"
  }

  actor "User" {
    auth_required = false
  }
}
