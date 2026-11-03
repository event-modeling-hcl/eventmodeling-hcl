# Clinic staff register pets through a client program that calls the clinic
# API, and read the pet directory through the same API. Each API request and
# each API response is a screen with kind = "api".

board {
  workflows = [
    workflow.register_pet,
    workflow.pet_directory,
  ]
}

actor "clinic_staff" {
  title         = "Clinic Staff"
  auth_required = true
}

bounded_context "clinic" {
  title = "Clinic"

  aggregate "pet" {
    title = "Pet"
  }

  field_type "pet_id" {
    type         = "UUID"
    id_attribute = true
  }

  field_type "pet_name" {
    type = "String"
  }

  event "pet_registered" {
    title     = "Pet Registered"
    aggregate = aggregate.pet

    field "pet_id" {
      type = field_type.clinic.pet_id
    }

    field "name" {
      type = field_type.clinic.pet_name
    }
  }
}

state_change "register_pet" {
  title = "Register Pet"

  screen "register_pet_request" {
    kind         = "api"
    api_endpoint = "POST /pets"
    actor        = actor.clinic_staff
    to           = [command.register_pet]

    field "name" {
      type = field_type.clinic.pet_name
    }
  }

  command "register_pet" {
    title     = "Register Pet"
    aggregate = aggregate.clinic.pet
    to        = [event.clinic.pet_registered]

    field "pet_id" {
      type      = field_type.clinic.pet_id
      generated = true
    }

    field "name" {
      type = field_type.clinic.pet_name
    }
  }

  scenario "pet_is_registered" {
    when {
      command = command.register_pet
    }

    then {
      event = event.clinic.pet_registered
    }
  }
}

state_view "pet_directory" {
  title = "Pet Directory"

  readmodel "pets" {
    question = "Which pets does the clinic know?"
    from     = [event.clinic.pet_registered]
    to       = [screen.pets_response]

    field "pet_id" {
      type = field_type.clinic.pet_id
    }

    field "name" {
      type = field_type.clinic.pet_name
    }
  }

  screen "pets_response" {
    kind         = "api"
    api_endpoint = "GET /pets"
    actor        = actor.clinic_staff

    field "pet_id" {
      type = field_type.clinic.pet_id
    }

    field "name" {
      type = field_type.clinic.pet_name
    }
  }

  scenario "registered_pet_is_listed" {
    given {
      event = event.clinic.pet_registered
    }

    then {
      readmodel = readmodel.pets
    }
  }
}
