slice "example" {
  title      = "Example"
  status     = "Created"
  index      = 1
  slice_type = "STATE_CHANGE"
  aggregates = ["Pet"]

  command "command" {
    title             = "Command"
    type              = "COMMAND"
    tags              = ["write"]
    context           = "INTERNAL"
    aggregate         = "Pet"
    creates_aggregate = true
    prototype         = { route = "/pets" }

    field "pet_id" {
      type         = "Int"
      example      = { value = 5 }
      id_attribute = true
      cardinality  = "Single"

      subfield "display_name" {
        type    = "String"
        example = "Betty"
      }
    }

    dependency "evt-pet-created" {
      type         = "OUTBOUND"
      title        = "Pet Created"
      element_type = "EVENT"
    }
  }

  table "pets" {
    title = "Pets"

    field "name" {
      type    = "String"
      example = "Betty"
    }
  }

  specification "create-pet" {
    title     = "Create pet"
    linked_id = "command"
    vertical  = true

    when "submit" {
      title     = "Command"
      type      = "SPEC_COMMAND"
      index     = 0
      spec_row  = 0
      linked_id = "command"
      examples  = [{ case = "happy" }]

      field "pet_id" {
        type    = "Int"
        example = 5
      }
    }

    then "created" {
      title     = "Pet Created"
      type      = "SPEC_EVENT"
      linked_id = "evt-pet-created"
    }

    comment {
      description = "Creates a pet."
    }
  }
}
