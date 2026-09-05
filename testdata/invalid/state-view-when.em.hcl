bounded_context "example" {
  event "pet_created" {}
}

state_view "view_pets" {
  readmodel "pets" {
    question = "Which pets exist?"
  }

  scenario "loaded" {
    given { event = event.example.pet_created }
    when { readmodel = readmodel.pets }
    then { readmodel = readmodel.pets }
  }
}
