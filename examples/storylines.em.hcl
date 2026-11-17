# Storylines group the chapters of a model. The board still sets the story
# order: the storyline "registration" comes before "care" because its chapter
# holds the first workflow of the board, although "care" comes first in ID
# order. The chapters of "care" follow its chapters list.

board {
  workflows = [
    workflow.register_pet,
    workflow.pet_directory,
    workflow.book_visit,
    workflow.treat_pet,
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

  aggregate "visit" {
    title = "Visit"
  }

  field_type "pet_id" {
    type         = "UUID"
    id_attribute = true
    example      = "07917485-4a6b-4e4e-8920-92ce6062a129"
  }

  field_type "pet_name" {
    type    = "String"
    example = "Mochi"
  }

  field_type "visit_id" {
    type         = "UUID"
    id_attribute = true
    example      = "5c1f4e0a-2b1d-4a59-9a43-0d6f1b2c3e4f"
  }

  field_type "treatment" {
    type    = "String"
    example = "Vaccination"
  }

  event "pet_registered" {
    title     = "Pet Registered"
    aggregate = aggregate.pet
    fields    = [field_type.pet_id, field_type.pet_name]
  }

  event "visit_booked" {
    title     = "Visit Booked"
    aggregate = aggregate.visit
    fields    = [field_type.visit_id, field_type.pet_id]
  }

  event "pet_treated" {
    title     = "Pet Treated"
    aggregate = aggregate.visit
    fields    = [field_type.visit_id, field_type.treatment]
  }
}

state_change "register_pet" {
  title = "Register Pet"

  screen "pet_form" {
    title  = "Pet Form"
    actor  = actor.clinic_staff
    fields = [field_type.clinic.pet_name]
    to     = [command.register_pet]
  }

  command "register_pet" {
    title     = "Register Pet"
    aggregate = aggregate.clinic.pet
    fields    = [field_type.clinic.pet_id, field_type.clinic.pet_name]
    to        = [event.clinic.pet_registered]
  }
}

state_view "pet_directory" {
  title = "Pet Directory"

  readmodel "pets" {
    title    = "Pets"
    question = "Which pets does the clinic know?"
    fields   = [field_type.clinic.pet_id, field_type.clinic.pet_name]
    from     = [event.clinic.pet_registered]
    to       = [screen.directory]
  }

  screen "directory" {
    title = "Directory"
    actor = actor.clinic_staff
  }
}

state_change "book_visit" {
  title = "Book Visit"

  screen "visit_form" {
    title  = "Visit Form"
    actor  = actor.clinic_staff
    fields = [field_type.clinic.pet_id]
    to     = [command.book_visit]
  }

  command "book_visit" {
    title     = "Book Visit"
    aggregate = aggregate.clinic.visit
    fields    = [field_type.clinic.visit_id, field_type.clinic.pet_id]
    to        = [event.clinic.visit_booked]
  }
}

state_change "treat_pet" {
  title = "Treat Pet"

  screen "treatment_form" {
    title  = "Treatment Form"
    actor  = actor.clinic_staff
    fields = [field_type.clinic.visit_id, field_type.clinic.treatment]
    to     = [command.treat_pet]
  }

  command "treat_pet" {
    title     = "Treat Pet"
    aggregate = aggregate.clinic.visit
    fields    = [field_type.clinic.visit_id, field_type.clinic.treatment]
    to        = [event.clinic.pet_treated]
  }
}

chapter "pet_records" {
  title     = "Register a pet"
  workflows = [workflow.register_pet, workflow.pet_directory]
}

chapter "booking" {
  title     = "Book a visit"
  workflows = [workflow.book_visit]
}

chapter "treatment" {
  title     = "Treat the pet"
  workflows = [workflow.treat_pet]
}

storyline "registration" {
  title       = "Registration"
  description = "The clinic records a new pet."
  chapters    = [chapter.pet_records]
}

storyline "care" {
  title       = "Care"
  description = "The pet comes to the clinic for a visit."
  chapters    = [chapter.booking, chapter.treatment]
}
