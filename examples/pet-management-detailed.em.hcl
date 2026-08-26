# Native HCL port of config-pet-management-detailed.json.
# Source IDs and linked IDs are retained verbatim.

slice "slice-pet-001" {
  aggregates = ["Owner"]
  context = "Display owner information including their pets list before adding or editing pets"
  index = 1
  slice_type = "STATE_VIEW"
  status = "Created"
  title = "View Owner Details with Pets"
  readmodel "rm-pet-001" {
    aggregate = "Owner"
    description = "org.springframework.samples.petclinic.owner.OwnerRepository.findById, Owner.getPets()"
    title = "Owner with Pets"
    type = "READMODEL"
    field "id" {
      example = "1"
      id_attribute = true
      optional = false
      type = "Int"
    }
    field "firstName" {
      example = "George"
      optional = false
      type = "String"
    }
    field "lastName" {
      example = "Franklin"
      optional = false
      type = "String"
    }
    field "address" {
      example = "110 W. Liberty St."
      optional = false
      type = "String"
    }
    field "city" {
      example = "Madison"
      optional = false
      type = "String"
    }
    field "telephone" {
      example = "6085551023"
      optional = false
      type = "String"
    }
    field "pets" {
      cardinality = "List"
      example = {"birthDate":"2010-09-07","id":1,"name":"Leo","type":"cat"}
      optional = false
      type = "Custom"
      subfield "id" {
        example = "1"
        id_attribute = true
        type = "Int"
      }
      subfield "name" {
        example = "Leo"
        type = "String"
      }
      subfield "birthDate" {
        example = "2010-09-07"
        type = "Date"
      }
      subfield "type" {
        example = "cat"
        type = "String"
      }
    }
    dependency "scr-pet-001" {
      element_type = "SCREEN"
      title = "Owner Details Screen"
      type = "OUTBOUND"
    }
  }
  screen "scr-pet-001" {
    aggregate = "Owner"
    description = "owners/ownerDetails.html - displays owner info with pets table"
    title = "Owner Details Screen"
    type = "SCREEN"
    field "ownerName" {
      example = "George Franklin"
      type = "String"
    }
    field "address" {
      example = "110 W. Liberty St."
      type = "String"
    }
    field "city" {
      example = "Madison"
      type = "String"
    }
    field "telephone" {
      example = "6085551023"
      type = "String"
    }
    field "petsList" {
      cardinality = "List"
      example = {"birthDate":"2010-09-07","name":"Leo","type":"cat"}
      type = "Custom"
    }
    dependency "rm-pet-001" {
      element_type = "READMODEL"
      title = "Owner with Pets"
      type = "INBOUND"
    }
  }
  specification "spec-pet-001" {
    linked_id = "slice-pet-001"
    slice_name = "View Owner Details with Pets"
    title = "Owner with existing pets displayed"
    given "spec-pet-001-g1" {
      index = 0
      linked_id = "evt-001"
      spec_row = 0
      title = "Owner Registered"
      type = "SPEC_EVENT"
      field "ownerId" {
        example = "1"
        type = "Int"
      }
      field "firstName" {
        example = "George"
        type = "String"
      }
      field "lastName" {
        example = "Franklin"
        type = "String"
      }
    }
    given "spec-pet-001-g2" {
      index = 1
      linked_id = "evt-pet-001"
      spec_row = 1
      title = "Pet Added"
      type = "SPEC_EVENT"
      field "petId" {
        example = "1"
        type = "Int"
      }
      field "name" {
        example = "Leo"
        type = "String"
      }
      field "birthDate" {
        example = "2010-09-07"
        type = "Date"
      }
      field "typeId" {
        example = "1"
        type = "Int"
      }
      field "ownerId" {
        example = "1"
        type = "Int"
      }
    }
    then "spec-pet-001-t1" {
      index = 0
      linked_id = "rm-pet-001"
      spec_row = 0
      title = "Owner with Pets"
      type = "SPEC_READMODEL"
      field "id" {
        example = "1"
        type = "Int"
      }
      field "firstName" {
        example = "George"
        type = "String"
      }
      field "pets" {
        cardinality = "List"
        example = {"birthDate":"2010-09-07","id":1,"name":"Leo"}
        type = "Custom"
      }
    }
  }
  actor "Clinic Staff" {
    auth_required = false
  }
}

slice "slice-pet-002" {
  aggregates = ["Pet","Owner"]
  context = "Register a new pet for an existing owner with validation"
  index = 2
  slice_type = "STATE_CHANGE"
  status = "Created"
  title = "Add New Pet to Owner"
  command "cmd-pet-001" {
    aggregate = "Pet"
    aggregate_dependencies = ["Owner"]
    api_endpoint = "POST /owners/{ownerId}/pets/new"
    creates_aggregate = true
    description = "org.springframework.samples.petclinic.owner.PetController.processCreationForm"
    title = "Add Pet"
    type = "COMMAND"
    field "name" {
      example = "Betty"
      optional = false
      type = "String"
    }
    field "birthDate" {
      example = "2015-02-12"
      optional = false
      type = "Date"
    }
    field "typeId" {
      example = "3"
      optional = false
      type = "Int"
    }
    field "ownerId" {
      example = "1"
      optional = false
      technical_attribute = true
      type = "Int"
    }
    dependency "scr-pet-002" {
      element_type = "SCREEN"
      title = "Pet Registration Form"
      type = "INBOUND"
    }
    dependency "evt-pet-001" {
      element_type = "EVENT"
      title = "Pet Added"
      type = "OUTBOUND"
    }
  }
  event "evt-pet-001" {
    aggregate = "Pet"
    aggregate_dependencies = ["Owner"]
    description = "org.springframework.samples.petclinic.owner.PetRepository.save, Owner.addPet"
    title = "Pet Added"
    type = "EVENT"
    field "petId" {
      example = "5"
      generated = true
      id_attribute = true
      type = "Int"
    }
    field "name" {
      example = "Betty"
      type = "String"
    }
    field "birthDate" {
      example = "2015-02-12"
      type = "Date"
    }
    field "typeId" {
      example = "3"
      type = "Int"
    }
    field "typeName" {
      example = "hamster"
      type = "String"
    }
    field "ownerId" {
      example = "1"
      type = "Int"
    }
    dependency "cmd-pet-001" {
      element_type = "COMMAND"
      title = "Add Pet"
      type = "INBOUND"
    }
  }
  screen "scr-pet-002" {
    aggregate = "Pet"
    description = "pets/createOrUpdatePetForm.html - form with title 'New Pet'"
    title = "Pet Registration Form"
    type = "SCREEN"
    field "ownerName" {
      example = "George Franklin"
      type = "String"
    }
    field "name" {
      example = "Betty"
      type = "String"
    }
    field "birthDate" {
      example = "2015-02-12"
      type = "Date"
    }
    field "type" {
      example = "hamster"
      type = "String"
    }
    field "availableTypes" {
      cardinality = "List"
      example = {"id":1,"name":"cat"}
      type = "Custom"
      subfield "id" {
        example = "1"
        type = "Int"
      }
      subfield "name" {
        example = "cat"
        type = "String"
      }
    }
    dependency "rm-pet-001" {
      element_type = "READMODEL"
      title = "Owner with Pets"
      type = "INBOUND"
    }
    dependency "rm-pet-002" {
      element_type = "READMODEL"
      title = "Available Pet Types"
      type = "INBOUND"
    }
    dependency "cmd-pet-001" {
      element_type = "COMMAND"
      title = "Add Pet"
      type = "OUTBOUND"
    }
  }
  specification "spec-pet-002" {
    linked_id = "slice-pet-002"
    slice_name = "Add New Pet to Owner"
    title = "Successfully add new pet with valid data"
    given "spec-pet-002-g1" {
      index = 0
      linked_id = "evt-001"
      spec_row = 0
      title = "Owner Registered"
      type = "SPEC_EVENT"
      field "ownerId" {
        example = "1"
        type = "Int"
      }
      field "firstName" {
        example = "George"
        type = "String"
      }
      field "lastName" {
        example = "Franklin"
        type = "String"
      }
    }
    when "spec-pet-002-w1" {
      index = 0
      linked_id = "cmd-pet-001"
      spec_row = 0
      title = "Add Pet"
      type = "SPEC_COMMAND"
      field "name" {
        example = "Betty"
        type = "String"
      }
      field "birthDate" {
        example = "2015-02-12"
        type = "Date"
      }
      field "typeId" {
        example = "3"
        type = "Int"
      }
      field "ownerId" {
        example = "1"
        type = "Int"
      }
    }
    then "spec-pet-002-t1" {
      index = 0
      linked_id = "evt-pet-001"
      spec_row = 0
      title = "Pet Added"
      type = "SPEC_EVENT"
      field "petId" {
        example = "5"
        type = "Int"
      }
      field "name" {
        example = "Betty"
        type = "String"
      }
      field "birthDate" {
        example = "2015-02-12"
        type = "Date"
      }
      field "typeId" {
        example = "3"
        type = "Int"
      }
      field "ownerId" {
        example = "1"
        type = "Int"
      }
    }
  }
  specification "spec-pet-003" {
    linked_id = "slice-pet-002"
    slice_name = "Add New Pet to Owner"
    title = "Reject pet with missing name"
    given "spec-pet-003-g1" {
      index = 0
      linked_id = "evt-001"
      spec_row = 0
      title = "Owner Registered"
      type = "SPEC_EVENT"
      field "ownerId" {
        example = "1"
        type = "Int"
      }
    }
    when "spec-pet-003-w1" {
      index = 0
      linked_id = "cmd-pet-001"
      spec_row = 0
      title = "Add Pet"
      type = "SPEC_COMMAND"
      field "name" {
        example = ""
        type = "String"
      }
      field "birthDate" {
        example = "2015-02-12"
        type = "Date"
      }
      field "typeId" {
        example = "3"
        type = "Int"
      }
    }
    then "spec-pet-003-t1" {
      index = 0
      linked_id = "cmd-pet-001"
      spec_row = 0
      title = "Validation Error"
      type = "SPEC_ERROR"
      field "field" {
        example = "name"
        type = "String"
      }
      field "errorCode" {
        example = "required"
        type = "String"
      }
    }
    comment {
      description = "PetValidator validates that name field has length - org.springframework.samples.petclinic.owner.PetValidator:39"
    }
  }
  specification "spec-pet-004" {
    linked_id = "slice-pet-002"
    slice_name = "Add New Pet to Owner"
    title = "Reject pet with missing type for new pets"
    given "spec-pet-004-g1" {
      index = 0
      linked_id = "evt-001"
      spec_row = 0
      title = "Owner Registered"
      type = "SPEC_EVENT"
      field "ownerId" {
        example = "1"
        type = "Int"
      }
    }
    when "spec-pet-004-w1" {
      index = 0
      linked_id = "cmd-pet-001"
      spec_row = 0
      title = "Add Pet"
      type = "SPEC_COMMAND"
      field "name" {
        example = "Betty"
        type = "String"
      }
      field "birthDate" {
        example = "2015-02-12"
        type = "Date"
      }
      field "typeId" {
        example = "null"
        type = "Int"
      }
    }
    then "spec-pet-004-t1" {
      index = 0
      linked_id = "cmd-pet-001"
      spec_row = 0
      title = "Validation Error"
      type = "SPEC_ERROR"
      field "field" {
        example = "type"
        type = "String"
      }
      field "errorCode" {
        example = "required"
        type = "String"
      }
    }
    comment {
      description = "PetValidator checks if pet.isNew and pet.type == null - org.springframework.samples.petclinic.owner.PetValidator:44"
    }
  }
  specification "spec-pet-005" {
    linked_id = "slice-pet-002"
    slice_name = "Add New Pet to Owner"
    title = "Reject pet with missing birth date"
    given "spec-pet-005-g1" {
      index = 0
      linked_id = "evt-001"
      spec_row = 0
      title = "Owner Registered"
      type = "SPEC_EVENT"
      field "ownerId" {
        example = "1"
        type = "Int"
      }
    }
    when "spec-pet-005-w1" {
      index = 0
      linked_id = "cmd-pet-001"
      spec_row = 0
      title = "Add Pet"
      type = "SPEC_COMMAND"
      field "name" {
        example = "Betty"
        type = "String"
      }
      field "birthDate" {
        example = "null"
        type = "Date"
      }
      field "typeId" {
        example = "3"
        type = "Int"
      }
    }
    then "spec-pet-005-t1" {
      index = 0
      linked_id = "cmd-pet-001"
      spec_row = 0
      title = "Validation Error"
      type = "SPEC_ERROR"
      field "field" {
        example = "birthDate"
        type = "String"
      }
      field "errorCode" {
        example = "required"
        type = "String"
      }
    }
    comment {
      description = "PetValidator validates that birthDate is not null - org.springframework.samples.petclinic.owner.PetValidator:49"
    }
  }
  specification "spec-pet-006" {
    linked_id = "slice-pet-002"
    slice_name = "Add New Pet to Owner"
    title = "Reject duplicate pet name for same owner"
    given "spec-pet-006-g1" {
      index = 0
      linked_id = "evt-001"
      spec_row = 0
      title = "Owner Registered"
      type = "SPEC_EVENT"
      field "ownerId" {
        example = "1"
        type = "Int"
      }
    }
    given "spec-pet-006-g2" {
      index = 1
      linked_id = "evt-pet-001"
      spec_row = 1
      title = "Pet Added"
      type = "SPEC_EVENT"
      field "petId" {
        example = "1"
        type = "Int"
      }
      field "name" {
        example = "Leo"
        type = "String"
      }
      field "ownerId" {
        example = "1"
        type = "Int"
      }
    }
    when "spec-pet-006-w1" {
      index = 0
      linked_id = "cmd-pet-001"
      spec_row = 0
      title = "Add Pet"
      type = "SPEC_COMMAND"
      field "name" {
        example = "Leo"
        type = "String"
      }
      field "birthDate" {
        example = "2020-01-15"
        type = "Date"
      }
      field "typeId" {
        example = "2"
        type = "Int"
      }
      field "ownerId" {
        example = "1"
        type = "Int"
      }
    }
    then "spec-pet-006-t1" {
      index = 0
      linked_id = "cmd-pet-001"
      spec_row = 0
      title = "Validation Error"
      type = "SPEC_ERROR"
      field "field" {
        example = "name"
        type = "String"
      }
      field "errorCode" {
        example = "duplicate"
        type = "String"
      }
      field "message" {
        example = "already exists"
        type = "String"
      }
    }
    comment {
      description = "PetController checks if owner.getPet(pet.name, true) != null for new pets - org.springframework.samples.petclinic.owner.PetController:67. Note: comparison is case-insensitive - Owner.getPet:81"
    }
  }
  actor "Clinic Staff" {
    auth_required = false
  }
}

slice "slice-pet-003" {
  aggregates = ["PetType"]
  context = "Display list of all available pet types for selection in forms"
  index = 3
  slice_type = "STATE_VIEW"
  status = "Created"
  title = "View Available Pet Types"
  readmodel "rm-pet-002" {
    aggregate = "PetType"
    description = "org.springframework.samples.petclinic.owner.PetRepository.findPetTypes"
    title = "Available Pet Types"
    type = "READMODEL"
    field "types" {
      cardinality = "List"
      example = {"id":1,"name":"cat"}
      type = "Custom"
      subfield "id" {
        example = "1"
        id_attribute = true
        type = "Int"
      }
      subfield "name" {
        example = "cat"
        type = "String"
      }
    }
    dependency "scr-pet-002" {
      element_type = "SCREEN"
      title = "Pet Registration Form"
      type = "OUTBOUND"
    }
    dependency "scr-pet-003" {
      element_type = "SCREEN"
      title = "Pet Edit Form"
      type = "OUTBOUND"
    }
  }
  specification "spec-pet-007" {
    linked_id = "slice-pet-003"
    slice_name = "View Available Pet Types"
    title = "Display all pet types sorted by name"
    then "spec-pet-007-t1" {
      index = 0
      linked_id = "rm-pet-002"
      spec_row = 0
      title = "Available Pet Types"
      type = "SPEC_READMODEL"
      field "types" {
        cardinality = "List"
        example = [{"id":2,"name":"bird"},{"id":1,"name":"cat"},{"id":3,"name":"dog"},{"id":4,"name":"hamster"},{"id":5,"name":"lizard"},{"id":6,"name":"snake"}]
        type = "Custom"
      }
    }
    comment {
      description = "PetRepository query orders types by name ascending - @Query('SELECT ptype FROM PetType ptype ORDER BY ptype.name')"
    }
  }
  actor "Clinic Staff" {
    auth_required = false
  }
}

slice "slice-pet-004" {
  aggregates = ["Pet"]
  context = "Modify existing pet details with validation"
  index = 4
  slice_type = "STATE_CHANGE"
  status = "Created"
  title = "Update Pet Information"
  command "cmd-pet-002" {
    aggregate = "Pet"
    api_endpoint = "POST /owners/{ownerId}/pets/{petId}/edit"
    description = "org.springframework.samples.petclinic.owner.PetController.processUpdateForm"
    title = "Update Pet"
    type = "COMMAND"
    field "petId" {
      example = "1"
      id_attribute = true
      optional = false
      type = "Int"
    }
    field "name" {
      example = "Betty"
      optional = false
      type = "String"
    }
    field "birthDate" {
      example = "2015-02-12"
      optional = false
      type = "Date"
    }
    field "typeId" {
      example = "3"
      optional = false
      type = "Int"
    }
    field "ownerId" {
      example = "1"
      optional = false
      technical_attribute = true
      type = "Int"
    }
    dependency "scr-pet-003" {
      element_type = "SCREEN"
      title = "Pet Edit Form"
      type = "INBOUND"
    }
    dependency "evt-pet-002" {
      element_type = "EVENT"
      title = "Pet Updated"
      type = "OUTBOUND"
    }
  }
  event "evt-pet-002" {
    aggregate = "Pet"
    description = "org.springframework.samples.petclinic.owner.PetRepository.save"
    title = "Pet Updated"
    type = "EVENT"
    field "petId" {
      example = "1"
      id_attribute = true
      type = "Int"
    }
    field "name" {
      example = "Betty"
      type = "String"
    }
    field "birthDate" {
      example = "2015-02-12"
      type = "Date"
    }
    field "typeId" {
      example = "3"
      type = "Int"
    }
    field "typeName" {
      example = "hamster"
      type = "String"
    }
    field "ownerId" {
      example = "1"
      type = "Int"
    }
    dependency "cmd-pet-002" {
      element_type = "COMMAND"
      title = "Update Pet"
      type = "INBOUND"
    }
  }
  screen "scr-pet-003" {
    aggregate = "Pet"
    description = "pets/createOrUpdatePetForm.html - form with title 'Pet'"
    title = "Pet Edit Form"
    type = "SCREEN"
    field "petId" {
      example = "1"
      type = "Int"
    }
    field "ownerName" {
      example = "George Franklin"
      type = "String"
    }
    field "name" {
      example = "Betty"
      type = "String"
    }
    field "birthDate" {
      example = "2015-02-12"
      type = "Date"
    }
    field "type" {
      example = "hamster"
      type = "String"
    }
    field "availableTypes" {
      cardinality = "List"
      example = {"id":1,"name":"cat"}
      type = "Custom"
      subfield "id" {
        example = "1"
        type = "Int"
      }
      subfield "name" {
        example = "cat"
        type = "String"
      }
    }
    dependency "rm-pet-003" {
      element_type = "READMODEL"
      title = "Pet Details"
      type = "INBOUND"
    }
    dependency "rm-pet-002" {
      element_type = "READMODEL"
      title = "Available Pet Types"
      type = "INBOUND"
    }
    dependency "cmd-pet-002" {
      element_type = "COMMAND"
      title = "Update Pet"
      type = "OUTBOUND"
    }
  }
  specification "spec-pet-008" {
    linked_id = "slice-pet-004"
    slice_name = "Update Pet Information"
    title = "Successfully update existing pet"
    given "spec-pet-008-g1" {
      index = 0
      linked_id = "evt-pet-001"
      spec_row = 0
      title = "Pet Added"
      type = "SPEC_EVENT"
      field "petId" {
        example = "1"
        type = "Int"
      }
      field "name" {
        example = "Leo"
        type = "String"
      }
      field "birthDate" {
        example = "2010-09-07"
        type = "Date"
      }
      field "typeId" {
        example = "1"
        type = "Int"
      }
    }
    when "spec-pet-008-w1" {
      index = 0
      linked_id = "cmd-pet-002"
      spec_row = 0
      title = "Update Pet"
      type = "SPEC_COMMAND"
      field "petId" {
        example = "1"
        type = "Int"
      }
      field "name" {
        example = "Betty"
        type = "String"
      }
      field "birthDate" {
        example = "2015-02-12"
        type = "Date"
      }
      field "typeId" {
        example = "3"
        type = "Int"
      }
    }
    then "spec-pet-008-t1" {
      index = 0
      linked_id = "evt-pet-002"
      spec_row = 0
      title = "Pet Updated"
      type = "SPEC_EVENT"
      field "petId" {
        example = "1"
        type = "Int"
      }
      field "name" {
        example = "Betty"
        type = "String"
      }
      field "birthDate" {
        example = "2015-02-12"
        type = "Date"
      }
      field "typeId" {
        example = "3"
        type = "Int"
      }
    }
  }
  specification "spec-pet-009" {
    linked_id = "slice-pet-004"
    slice_name = "Update Pet Information"
    title = "Reject update with missing name"
    given "spec-pet-009-g1" {
      index = 0
      linked_id = "evt-pet-001"
      spec_row = 0
      title = "Pet Added"
      type = "SPEC_EVENT"
      field "petId" {
        example = "1"
        type = "Int"
      }
      field "name" {
        example = "Leo"
        type = "String"
      }
    }
    when "spec-pet-009-w1" {
      index = 0
      linked_id = "cmd-pet-002"
      spec_row = 0
      title = "Update Pet"
      type = "SPEC_COMMAND"
      field "petId" {
        example = "1"
        type = "Int"
      }
      field "name" {
        example = ""
        type = "String"
      }
      field "birthDate" {
        example = "2015-02-12"
        type = "Date"
      }
      field "typeId" {
        example = "3"
        type = "Int"
      }
    }
    then "spec-pet-009-t1" {
      index = 0
      linked_id = "cmd-pet-002"
      spec_row = 0
      title = "Validation Error"
      type = "SPEC_ERROR"
      field "field" {
        example = "name"
        type = "String"
      }
      field "errorCode" {
        example = "required"
        type = "String"
      }
    }
    comment {
      description = "PetValidator validates that name field has length - org.springframework.samples.petclinic.owner.PetValidator:39"
    }
  }
  specification "spec-pet-010" {
    linked_id = "slice-pet-004"
    slice_name = "Update Pet Information"
    title = "Reject update with missing birth date"
    given "spec-pet-010-g1" {
      index = 0
      linked_id = "evt-pet-001"
      spec_row = 0
      title = "Pet Added"
      type = "SPEC_EVENT"
      field "petId" {
        example = "1"
        type = "Int"
      }
    }
    when "spec-pet-010-w1" {
      index = 0
      linked_id = "cmd-pet-002"
      spec_row = 0
      title = "Update Pet"
      type = "SPEC_COMMAND"
      field "petId" {
        example = "1"
        type = "Int"
      }
      field "name" {
        example = "Betty"
        type = "String"
      }
      field "birthDate" {
        example = "null"
        type = "Date"
      }
      field "typeId" {
        example = "3"
        type = "Int"
      }
    }
    then "spec-pet-010-t1" {
      index = 0
      linked_id = "cmd-pet-002"
      spec_row = 0
      title = "Validation Error"
      type = "SPEC_ERROR"
      field "field" {
        example = "birthDate"
        type = "String"
      }
      field "errorCode" {
        example = "required"
        type = "String"
      }
    }
    comment {
      description = "PetValidator validates that birthDate is not null - org.springframework.samples.petclinic.owner.PetValidator:49"
    }
  }
  actor "Clinic Staff" {
    auth_required = false
  }
}

slice "slice-pet-005" {
  aggregates = ["Pet"]
  context = "Load existing pet information to populate edit form"
  index = 5
  slice_type = "STATE_VIEW"
  status = "Created"
  title = "View Pet Details for Editing"
  readmodel "rm-pet-003" {
    aggregate = "Pet"
    description = "org.springframework.samples.petclinic.owner.PetRepository.findById"
    title = "Pet Details"
    type = "READMODEL"
    field "id" {
      example = "1"
      id_attribute = true
      type = "Int"
    }
    field "name" {
      example = "Leo"
      type = "String"
    }
    field "birthDate" {
      example = "2010-09-07"
      type = "Date"
    }
    field "typeId" {
      example = "1"
      type = "Int"
    }
    field "typeName" {
      example = "cat"
      type = "String"
    }
    field "ownerId" {
      example = "1"
      type = "Int"
    }
    dependency "evt-pet-001" {
      element_type = "EVENT"
      title = "Pet Added"
      type = "INBOUND"
    }
    dependency "evt-pet-002" {
      element_type = "EVENT"
      title = "Pet Updated"
      type = "INBOUND"
    }
    dependency "scr-pet-003" {
      element_type = "SCREEN"
      title = "Pet Edit Form"
      type = "OUTBOUND"
    }
  }
  specification "spec-pet-011" {
    linked_id = "slice-pet-005"
    slice_name = "View Pet Details for Editing"
    title = "Load existing pet for editing"
    given "spec-pet-011-g1" {
      index = 0
      linked_id = "evt-pet-001"
      spec_row = 0
      title = "Pet Added"
      type = "SPEC_EVENT"
      field "petId" {
        example = "1"
        type = "Int"
      }
      field "name" {
        example = "Leo"
        type = "String"
      }
      field "birthDate" {
        example = "2010-09-07"
        type = "Date"
      }
      field "typeId" {
        example = "1"
        type = "Int"
      }
    }
    then "spec-pet-011-t1" {
      index = 0
      linked_id = "rm-pet-003"
      spec_row = 0
      title = "Pet Details"
      type = "SPEC_READMODEL"
      field "id" {
        example = "1"
        type = "Int"
      }
      field "name" {
        example = "Leo"
        type = "String"
      }
      field "birthDate" {
        example = "2010-09-07"
        type = "Date"
      }
      field "typeId" {
        example = "1"
        type = "Int"
      }
    }
  }
  actor "Clinic Staff" {
    auth_required = false
  }
}
