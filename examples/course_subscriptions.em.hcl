# Complete executable reference for Event Modeling HCL v1.
# It is intentionally compact while covering every mapped property and enum.

slice "register-student" {
  title      = "Register student"
  status     = "Created"
  index      = 1
  context    = "Clinic"
  slice_type = "STATE_CHANGE"
  aggregates = ["Pet", "Owner"]

  command "register-pet" {
    group_id               = "registration"
    tags                   = ["write", "pet"]
    domain                 = "Pet management"
    model_context          = "Clinic operations"
    context                = "INTERNAL"
    slice                  = "pet-lifecycle"
    title                  = "Register pet"
    type                   = "COMMAND"
    description            = "Records a pet."
    aggregate              = "Pet"
    aggregate_dependencies = ["Owner"]
    api_endpoint           = "POST /pets"
    service                = null
    creates_aggregate      = true
    triggers               = ["submit"]
    sketched               = false
    prototype              = { route = "/pets", method = "POST" }
    list_element           = false

    field "name" {
      type                = "String"
      example             = "Mochi"
      mapping             = "pet.name"
      optional            = false
      technical_attribute = false
      generated           = false
      id_attribute        = false
      pii                 = true
      schema              = "PetName"
      cardinality         = "Single"
    }

    field "vaccinated" {
      type    = "Boolean"
      example = true
    }
    field "weight" {
      type    = "Double"
      example = 4.2
    }
    field "fee" {
      type    = "Decimal"
      example = 12.50
    }
    field "microchip" {
      type    = "Long"
      example = 1234567890
    }
    field "birth_date" {
      type    = "Date"
      example = "2020-01-12"
    }
    field "registered_at" {
      type    = "DateTime"
      example = "2025-01-12T10:30:00Z"
    }
    field "external_id" {
      type    = "UUID"
      example = "07917485-4a6b-4e4e-8920-92ce6062a129"
    }
    field "pet_id" {
      type         = "Int"
      example      = 5
      id_attribute = true
    }
    field "unset_value" {
      type    = "Custom"
      example = null
    }

    field "pets" {
      type        = "Custom"
      cardinality = "List"
      example = [{
        id   = 5
        name = "Mochi"
      }]

      subfield "id" {
        type         = "Int"
        example      = 5
        id_attribute = true
      }

      subfield "owner" {
        type = "Custom"
        example = {
          name = "Sam"
        }
        subfield "name" {
          type    = "String"
          example = "Sam"
        }
      }
    }

    dependency "evt-pet-registered" {
      type         = "OUTBOUND"
      title        = "Pet registered"
      element_type = "EVENT"
    }
  }

  event "evt-pet-registered" {
    title        = "Pet registered"
    type         = "EVENT"
    context      = "EXTERNAL"
    service      = "registration-service"
    list_element = true

    field "pet_id" {
      type    = "Int"
      example = 5
    }

    dependency "register-pet" {
      type         = "INBOUND"
      title        = "Register pet"
      element_type = "COMMAND"
    }
  }

  readmodel "pet-summary" {
    title = "Pet summary"
    type  = "READMODEL"

    dependency "pet-screen" {
      type         = "OUTBOUND"
      title        = "Pet screen"
      element_type = "SCREEN"
    }
  }

  screen "pet-screen" {
    title = "Pet screen"
    type  = "SCREEN"

    dependency "pet-notification" {
      type         = "OUTBOUND"
      title        = "Pet notification"
      element_type = "AUTOMATION"
    }
  }

  processor "pet-notification" {
    title = "Pet notification"
    type  = "AUTOMATION"
  }

  screen_image "pet-form" {
    title = "Pet form"
    url   = "https://example.test/pet-form.png"
  }

  table "pets" {
    title = "Pets"
    field "pet_id" {
      type    = "Int"
      example = 5
    }
  }

  specification "register-pet-specification" {
    vertical   = true
    title      = "Register a pet"
    slice_name = "Pet lifecycle"
    linked_id  = "register-pet"

    given "existing-owner" {
      title             = "Owner exists"
      tags              = ["setup"]
      examples          = [{ owner_id = 9 }]
      index             = 0
      spec_row          = 0
      type              = "SPEC_EVENT"
      linked_id         = "evt-owner-registered"
      expect_empty_list = false
    }

    when "register-pet" {
      title     = "Register pet"
      type      = "SPEC_COMMAND"
      linked_id = "register-pet"
      field "name" {
        type    = "String"
        example = "Mochi"
      }
    }

    then "pet-summary" {
      title     = "Pet summary updated"
      type      = "SPEC_READMODEL"
      linked_id = "pet-summary"
    }

    then "validation-error" {
      title             = "Validation error"
      type              = "SPEC_ERROR"
      linked_id         = "register-pet-error"
      expect_empty_list = true
    }

    comment {
      description = "The owner relationship is assumed to exist."
    }
  }

  actor "Clinic staff" {
    auth_required = true
  }
}

slice "pet-directory" {
  title      = "Pet directory"
  status     = "Done"
  index      = 2
  slice_type = "STATE_VIEW"
}

slice "notify-owner" {
  title      = "Notify owner"
  status     = "InProgress"
  index      = 3
  slice_type = "AUTOMATION"
}
