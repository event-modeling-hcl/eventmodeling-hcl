bounded_context "example" {
  event "pet_created" {}
}

state_change "create_pet" {
  screen "form" {
    to = [command.create]
  }

  command "create" {
    from = [screen.form]
    to   = [event.example.pet_created]
  }
}
