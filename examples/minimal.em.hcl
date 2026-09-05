# Minimal Event Modeling document with context-owned contracts.

bounded_context "pet_management" {
  title = "Pet Management"

  aggregate "pet" {
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

state_change "add_pet" {
  title = "Add Pet"

  screen "add_pet_form" {
    title = "Add Pet Form"
    to    = [command.add_pet_command]
  }

  command "add_pet_command" {
    title     = "Add Pet"
    aggregate = aggregate.pet_management.pet
    to        = [event.pet_management.pet_added]

    field "pet_id" {
      type = field_type.pet_management.pet_id
    }
  }
}
