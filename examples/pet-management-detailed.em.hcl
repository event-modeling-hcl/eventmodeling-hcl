# Pet-management model using bounded-context-owned events and reusable field types.

actor "clinic_staff" {
  title         = "Clinic staff"
  auth_required = true
}

bounded_context "owner_management" {
  title = "Owner Management"

  aggregate "owner" {
  }

  field_type "owner_id" {
    type         = "Int"
    id_attribute = true
    example      = 42
  }

  field_type "owner_name" {
    type    = "String"
    pii     = true
    example = "Darcy"
  }

  field_type "owner_email" {
    type     = "String"
    optional = true
    pii      = true
    example  = "darcy@example.test"
  }

  event "owner_registered" {
    title = "Owner Registered"

    field "owner_id" {
      type = field_type.owner_id
    }
  }
}

bounded_context "pet_management" {
  title = "Pet Management"

  aggregate "pet" {
  }

  field_type "pet_id" {
    type         = "Int"
    id_attribute = true
    example      = 5
  }

  field_type "pet_name" {
    type    = "String"
    pii     = true
    example = "Mochi"
  }

  field_type "birth_date" {
    type     = "Date"
    optional = true
    example  = "2020-01-12"
  }

  field_type "pet_type" {
    type    = "String"
    example = "Cat"
  }

  field_type "owner_pets" {
    cardinality = "List"
    type        = "Custom"
    example = [{
      id        = 5
      name      = "Mochi"
      birthDate = "2020-01-12"
      type      = "Cat"
    }]

    subfield "id" {
      type         = "Int"
      id_attribute = true
    }

    subfield "name" {
      type = "String"
    }

    subfield "birth_date" {
      type = "Date"
    }

    subfield "type" {
      type = "String"
    }
  }

  event "pet_added" {
    title                  = "Pet Added"
    aggregate              = aggregate.pet
    aggregate_dependencies = [aggregate.owner_management.owner]

    field "pet_id" {
      type = field_type.pet_id
    }

    field "pet_name" {
      type = field_type.pet_name
    }

    field "birth_date" {
      type = field_type.birth_date
    }

    field "pet_type" {
      type = field_type.pet_type
    }

    field "owner_id" {
      type = field_type.owner_management.owner_id
    }
  }

  event "pet_details_updated" {
    title     = "Pet Details Updated"
    aggregate = aggregate.pet

    field "pet_id" {
      type = field_type.pet_id
    }

    field "pet_name" {
      type = field_type.pet_name
    }

    field "birth_date" {
      type = field_type.birth_date
    }

    field "pet_type" {
      type = field_type.pet_type
    }
  }
}

bounded_context "pet_type_catalog" {
  title = "Pet Type Catalog"

  aggregate "pet_type" {
  }

  field_type "pet_type_id" {
    type         = "Int"
    id_attribute = true
    example      = 1
  }

  field_type "pet_type_name" {
    type    = "String"
    example = "Cat"
  }
}

state_view "show_owner_details" {
  title       = "Show Owner Details"
  description = "Display owner information including their pets list before adding or editing pets."

  readmodel "owner_details" {
    title    = "Owner Details"
    question = "What are the owner's details and registered pets?"
    from     = [event.pet_management.pet_added, event.pet_management.pet_details_updated]
    to       = [screen.owner_details_screen]

    field "owner_id" {
      type = field_type.owner_management.owner_id
    }

    field "owner_name" {
      type = field_type.owner_management.owner_name
    }

    field "owner_email" {
      type = field_type.owner_management.owner_email
    }

    field "owner_pets" {
      type = field_type.pet_management.owner_pets
    }
  }

  screen "owner_details_screen" {
    title = "Owner Details Screen"
    actor = actor.clinic_staff
  }

  scenario "owner_details_loaded" {
    title = "Owner details are displayed"

    given {
      title = "Pet Added"
      event = event.pet_management.pet_added
    }

    then {
      title     = "Owner Details"
      readmodel = readmodel.owner_details

      field "owner_id" {
        type = field_type.owner_management.owner_id
      }

      field "owner_pets" {
        type = field_type.pet_management.owner_pets
      }
    }
  }
}

state_change "add_pet" {
  title       = "Add Pet"
  description = "Register a new pet for an existing owner with validation."

  screen "add_pet_form" {
    title = "Add Pet Form"
    actor = actor.clinic_staff
    to    = [command.add_pet_command]
  }

  command "add_pet_command" {
    title                  = "Add Pet"
    aggregate              = aggregate.pet_management.pet
    aggregate_dependencies = [aggregate.owner_management.owner]
    api_endpoint           = "POST /owners/{ownerId}/pets"
    creates_aggregate      = true
    to                     = [event.pet_management.pet_added]

    field "owner_id" {
      type = field_type.owner_management.owner_id
    }

    field "pet_name" {
      type = field_type.pet_management.pet_name
    }

    field "birth_date" {
      type = field_type.pet_management.birth_date
    }

    field "pet_type" {
      type = field_type.pet_management.pet_type
    }
  }

  scenario "add_pet_success" {
    title = "Add pet successfully"

    given {
      title = "Owner exists"
      event = event.owner_management.owner_registered

      field "owner_id" {
        type = field_type.owner_management.owner_id
      }
    }

    when {
      title   = "Add Pet"
      command = command.add_pet_command

      field "pet_name" {
        type = field_type.pet_management.pet_name
      }

      field "birth_date" {
        type = field_type.pet_management.birth_date
      }

      field "pet_type" {
        type = field_type.pet_management.pet_type
      }
    }

    then {
      title = "Pet Added"
      event = event.pet_management.pet_added

      field "pet_id" {
        type = field_type.pet_management.pet_id
      }

      field "pet_name" {
        type = field_type.pet_management.pet_name
      }
    }
  }

  scenario "add_pet_validation_error" {
    title = "Reject missing pet name"

    when {
      title   = "Add Pet without a name"
      command = command.add_pet_command
    }

    then {
      title             = "Validation Error"
      expect_empty_list = true
      error             = "Pet name is required"
    }

    comment {
      description = "A pet must have a name before it can be registered."
    }
  }
}

state_view "list_pet_types" {
  title       = "List Pet Types"
  description = "Display list of all available pet types for selection in forms."

  readmodel "pet_type_list" {
    title    = "Pet Type List"
    question = "Which pet types can be selected?"
    from     = [event.pet_management.pet_added]
    to       = [screen.pet_type_picker]

    field "types" {
      cardinality = "List"
      type        = "Custom"
      example = [{
        id   = 1
        name = "Cat"
      }]

      subfield "id" {
        type         = "Int"
        id_attribute = true
      }

      subfield "name" {
        type = "String"
      }
    }
  }

  screen "pet_type_picker" {
    title = "Pet Type Picker"
    actor = actor.clinic_staff
  }

  scenario "pet_types_available" {
    title = "Pet types are available"

    given {
      title = "Pet Added"
      event = event.pet_management.pet_added
    }

    then {
      title     = "Pet Type List"
      readmodel = readmodel.pet_type_list
    }
  }
}

state_change "edit_pet" {
  title       = "Edit Pet"
  description = "Modify existing pet details with validation."

  screen "edit_pet_form" {
    title = "Edit Pet Form"
    actor = actor.clinic_staff
    to    = [command.edit_pet_command]
  }

  command "edit_pet_command" {
    title     = "Edit Pet"
    aggregate = aggregate.pet_management.pet
    to        = [event.pet_management.pet_details_updated]

    field "pet_id" {
      type = field_type.pet_management.pet_id
    }

    field "pet_name" {
      type = field_type.pet_management.pet_name
    }

    field "birth_date" {
      type = field_type.pet_management.birth_date
    }

    field "pet_type" {
      type = field_type.pet_management.pet_type
    }
  }

  scenario "edit_pet_success" {
    title = "Edit pet successfully"

    given {
      title = "Pet exists"
      event = event.pet_management.pet_added

      field "pet_id" {
        type = field_type.pet_management.pet_id
      }
    }

    when {
      title   = "Edit Pet"
      command = command.edit_pet_command

      field "pet_name" {
        type = field_type.pet_management.pet_name
      }
    }

    then {
      title = "Pet Details Updated"
      event = event.pet_management.pet_details_updated

      field "pet_id" {
        type = field_type.pet_management.pet_id
      }
    }
  }
}

state_view "load_pet_details" {
  title       = "Load Pet Details"
  description = "Load existing pet information to populate edit form."

  readmodel "pet_details" {
    title    = "Pet Details"
    question = "What are the current details of this pet?"
    from     = [event.pet_management.pet_added, event.pet_management.pet_details_updated]
    to       = [screen.pet_details_screen]

    field "pet_id" {
      type = field_type.pet_management.pet_id
    }

    field "pet_name" {
      type = field_type.pet_management.pet_name
    }

    field "birth_date" {
      type = field_type.pet_management.birth_date
    }

    field "pet_type" {
      type = field_type.pet_management.pet_type
    }
  }

  screen "pet_details_screen" {
    title = "Pet Details Screen"
    actor = actor.clinic_staff
  }

  scenario "pet_details_loaded" {
    title = "Pet details are loaded"

    given {
      title = "Pet Added"
      event = event.pet_management.pet_added
    }

    then {
      title     = "Pet Details"
      readmodel = readmodel.pet_details

      field "pet_id" {
        type = field_type.pet_management.pet_id
      }
    }
  }
}
