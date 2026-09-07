# Complete executable reference for Event Modeling HCL with bounded contexts.

team "clinic_team" {
  title = "Clinic team"
}

system "partner_system" {
  title    = "Partner system"
  external = true
}

actor "clinic_staff" {
  title         = "Clinic staff"
  auth_required = true
}

bounded_context "clinic" {
  title       = "Clinic"
  description = "Pet clinic operations and records."
  owner       = team.clinic_team

  aggregate "pet" {
    title       = "Pet"
    description = "A registered animal patient."
  }

  aggregate "owner" {
    title = "Owner"
  }

  field_type "pet_id" {
    cardinality  = "Single"
    type         = "UUID"
    id_attribute = true
    example      = "07917485-4a6b-4e4e-8920-92ce6062a129"
  }

  field_type "pet_name" {
    mapping             = "pet.name"
    schema              = "PetName"
    type                = "String"
    generated           = false
    id_attribute        = false
    optional            = false
    pii                 = true
    technical_attribute = false
    example             = "Mochi"
  }

  field_type "vaccinated" {
    type    = "Boolean"
    example = true
  }

  field_type "weight" {
    type    = "Double"
    example = 4.2
  }

  field_type "fee" {
    type    = "Decimal"
    example = 12.50
  }

  field_type "microchip" {
    type    = "Long"
    example = 1234567890
  }

  field_type "birth_date" {
    type    = "Date"
    example = "2020-01-12"
  }

  field_type "registered_at" {
    type    = "DateTime"
    example = "2025-01-12T10:30:00Z"
  }

  field_type "pet_profile" {
    cardinality = "List"
    type        = "Custom"
    example = [{
      id   = "07917485-4a6b-4e4e-8920-92ce6062a129"
      name = "Mochi"
    }]

    subfield "id" {
      type         = "UUID"
      id_attribute = true
    }

    subfield "name" {
      type = "String"
    }
  }

  event "pet_registered" {
    title                  = "Pet Registered"
    description            = "Records that a pet was registered."
    group_id               = "registration"
    tags                   = ["event", "pet"]
    aggregate              = aggregate.pet
    aggregate_dependencies = [aggregate.owner]
    service                = null
    list_element           = false
    prototype              = { topic = "clinic.pet_registered" }
    sketched               = false

    field "pet_id" {
      type = field_type.pet_id
    }

    field "pet_name" {
      type = field_type.pet_name
    }

    field "registered_at" {
      type = field_type.registered_at
    }
  }

  event "external_pet_imported" {
    field "pet_id" {
      type = field_type.pet_id
    }
  }

  event "owner_registered" {
  }
}

bounded_context "partner" {
  title    = "Partner"
  owner    = system.partner_system
  external = true

  event "pet_received" {
    title = "Pet Received"

    field "pet_id" {
      type = field_type.clinic.pet_id
    }
  }
}

state_change "register_pet" {
  description = "Register a new pet for an owner."
  owner       = bounded_context.clinic
  status      = "created"

  screen "pet_screen" {
    title  = "Pet screen"
    actor  = actor.clinic_staff
    fields = [field_type.clinic.pet_name]
    to     = [command.register_pet_command]

    field "pet_id" {
    }
  }

  screen_image "pet_form" {
    title = "Pet form"
    url   = "https://example.test/pet-form.png"
  }

  command "register_pet_command" {
    title                  = "Register pet"
    description            = "Records a pet."
    group_id               = "registration"
    tags                   = ["write", "pet"]
    aggregate              = aggregate.clinic.pet
    aggregate_dependencies = [aggregate.clinic.owner]
    api_endpoint           = "POST /pets"
    service                = null
    creates_aggregate      = true
    triggers               = ["submit"]
    list_element           = false
    prototype              = { route = "/pets", method = "POST" }
    sketched               = false
    to                     = [event.clinic.pet_registered]

    field "pet_id" {
      type = field_type.clinic.pet_id
    }

    field "pet_name" {
      type = field_type.clinic.pet_name
    }

    field "request_id" {
      type                = "UUID"
      technical_attribute = true
    }
  }

  table "pets" {
    title = "Pets"

    field "pet_id" {
      type = field_type.clinic.pet_id
    }
  }

  scenario "register_pet_specification" {
    title = "Register a pet"

    given {
      title             = "Owner exists"
      tags              = ["setup"]
      examples          = [{ owner_id = 9 }]
      expect_empty_list = false
      event             = event.clinic.owner_registered
    }

    when {
      title   = "Register pet"
      command = command.register_pet_command

      field "pet_name" {
        type = field_type.clinic.pet_name
      }
    }

    then {
      title = "Pet registered"
      event = event.clinic.pet_registered
    }

    then {
      title             = "Validation error"
      expect_empty_list = true
      error             = "Pet registration is invalid"
    }

    comment {
      description = "The owner relationship is assumed to exist."
    }
  }
}

state_view "pet_directory" {
  title  = "Pet directory"
  status = "done"

  readmodel "pet_summary" {
    title    = "Pet summary"
    question = "Which pets are registered?"
    from     = [event.clinic.pet_registered]
    to       = [screen.pet_summary_screen]

    field "pet_profile" {
      type = field_type.clinic.pet_profile
    }
  }

  screen "pet_summary_screen" {
    title = "Pet summary screen"
    actor = actor.clinic_staff
  }
}

automation "notify_owner" {
  title  = "Notify owner"
  status = "in_progress"

  readmodel "pets_needing_notification" {
    title    = "Pets needing notification"
    question = "Which pets require an owner notification?"
    from     = [event.clinic.pet_registered]
    to       = [processor.pet_notification]
  }

  processor "pet_notification" {
    title = "Pet notification"
    to    = [command.send_owner_notification]
  }

  command "send_owner_notification" {
    title     = "Send owner notification"
    aggregate = aggregate.clinic.owner
    to        = [event.clinic.external_pet_imported]
  }
}

translation "import_partner_pet" {
  title       = "Import partner pet"
  description = "Translate the partner contract into the clinic language."

  processor "translate_pet" {
    title = "Translate partner pet"
    from  = [event.partner.pet_received]
    to    = [command.import_pet]
  }

  command "import_pet" {
    title = "Import pet"
    to    = [event.clinic.external_pet_imported]
  }
}

chapter "registration" {
  title     = "Registration"
  workflows = [workflow.register_pet, workflow.pet_directory, workflow.notify_owner, workflow.import_partner_pet]
}

hotspot "notification_channel" {
  status   = "open"
  question = "Which notification channel should be used?"
  on       = processor.notify_owner.pet_notification
}
