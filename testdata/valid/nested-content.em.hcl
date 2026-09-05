bounded_context "example" {
  title = "Example"

  aggregate "pet" {
  }

  field_type "pet_id" {
    cardinality  = "Single"
    type         = "Int"
    id_attribute = true
    example      = 5

    subfield "display_name" {
      type    = "String"
      example = "Betty"
    }
  }

  event "pet_created" {
    title     = "Pet Created"
    aggregate = aggregate.pet

    field "pet_id" {
      type = field_type.pet_id
    }
  }
}

state_change "example" {
  title  = "Example"
  status = "created"

  command "command" {
    title             = "Command"
    tags              = ["write"]
    aggregate         = aggregate.example.pet
    creates_aggregate = true
    external_trigger  = true
    prototype         = { route = "/pets" }
    to                = [event.example.pet_created]

    field "pet_id" {
      type = field_type.example.pet_id
    }
  }

  table "pets" {
    title = "Pets"

    field "name" {
      type    = "String"
      example = "Betty"
    }
  }

  scenario "create_pet" {
    title = "Create pet"

    when {
      title    = "Command"
      examples = [{ case = "happy" }]
      command  = command.command

      field "pet_id" {
        type = field_type.example.pet_id
      }
    }

    then {
      title = "Pet Created"
      event = event.example.pet_created
    }

    comment {
      description = "Creates a pet."
    }
  }
}
