# Two elements in one workflow share the id "pet", which must be unique.
bounded_context "example" {
  title = "Example"
}

state_change "add_pet" {
  title = "Add Pet"

  command "pet" {
    title            = "Add Pet"
    external_trigger = true
  }

  command "pet" {
    title            = "Update Pet"
    external_trigger = true
  }
}
