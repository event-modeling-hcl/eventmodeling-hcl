board {
  workflows = [
    workflow.add_pet,
  ]
}

bounded_context "pet_management" {
  title = "Pet Management"

  aggregate "pet" {
    title = "Pet"
  }

  field_type "pet_id" {
    type         = "Int"
    id_attribute = true
    example      = 5
  }

  event "pet_added" {
    title     = "Pet Added"
    aggregate = aggregate.pet

    field "pet_id" {
      type = field_type.pet_id
    }
  }
}
