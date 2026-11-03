board {
  workflows = [workflow.register_pet, workflow.pet_directory, workflow.notify_owner]
}

team "reception" {}
team "communications" {}

actor "receptionist" {
  auth_required = true
}

bounded_context "clinic" {
  owner = team.reception

  aggregate "pet" {}

  field_type "pet_id" {
    type = "UUID"
  }

  field_type "owner_email" {
    type = "String"
    pii  = true
  }

  event "pet_registered" {
    aggregate = aggregate.pet
    fields    = [field_type.pet_id, field_type.owner_email]
  }

  event "owner_notice_requested" {
    fields = [field_type.pet_id, field_type.owner_email]
  }
}

chapter "registration" {
  workflows = [workflow.register_pet, workflow.pet_directory, workflow.notify_owner]
}

state_change "register_pet" {
  owner = team.reception

  screen "registration" {
    actor  = actor.receptionist
    to     = [command.register_pet]
    fields = [field_type.clinic.pet_id, field_type.clinic.owner_email]
  }

  command "register_pet" {
    aggregate = aggregate.clinic.pet
    to        = [event.clinic.pet_registered]
    fields    = [field_type.clinic.pet_id, field_type.clinic.owner_email]
  }

  scenario "pet_is_registered" {
    when { command = command.register_pet }
    then { event = event.clinic.pet_registered }
  }
}

state_view "pet_directory" {
  owner = team.reception

  readmodel "registered_pets" {
    question = "Which pets are registered at the clinic?"
    from     = [event.clinic.pet_registered]
    to       = [screen.pet_directory]
    fields   = [field_type.clinic.pet_id, field_type.clinic.owner_email]
  }

  screen "pet_directory" {
    actor = actor.receptionist
  }

  scenario "registered_pet_is_listed" {
    given { event = event.clinic.pet_registered }
    then { readmodel = readmodel.registered_pets }
  }
}

automation "notify_owner" {
  owner = team.communications

  readmodel "owners_needing_notice" {
    question = "Which registered pets still need an owner notice request?"
    from     = [event.clinic.pet_registered, event.clinic.owner_notice_requested]
    to       = [processor.owner_notifier]
    fields   = [field_type.clinic.pet_id, field_type.clinic.owner_email]
  }

  processor "owner_notifier" {
    to     = [command.request_owner_notice]
    fields = [field_type.clinic.pet_id, field_type.clinic.owner_email]
  }

  command "request_owner_notice" {
    to     = [event.clinic.owner_notice_requested]
    fields = [field_type.clinic.pet_id, field_type.clinic.owner_email]
  }

  scenario "notice_is_requested" {
    given { event = event.clinic.pet_registered }
    when { processor = processor.owner_notifier }
    then { event = event.clinic.owner_notice_requested }
  }

  scenario "requested_notice_leaves_the_list" {
    given { event = event.clinic.pet_registered }
    given { event = event.clinic.owner_notice_requested }
    then {
      readmodel         = readmodel.owners_needing_notice
      expect_empty_list = true
    }
  }
}

hotspot "owner_notice_timing" {
  question = "How soon after registration should the owner notice be requested?"
  status   = "open"
  on       = processor.notify_owner.owner_notifier
}
