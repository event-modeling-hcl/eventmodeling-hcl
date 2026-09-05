bounded_context "example" {
  title = "Example"

  aggregate "pet" {
  }

  field_type "pet_id" {
    type = "Int"
  }

  event "pet_created" {
    title     = "Pet Created"
    aggregate = aggregate.pet

    field "pet_id" {
      type = field_type.pet_id
    }
  }
}

actor "user" {
  title         = "User"
  auth_required = false
}

state_change "create_pet" {
  title = "Create Pet"

  command "command" {
    title            = "Command"
    aggregate        = aggregate.example.pet
    external_trigger = true
    to               = [event.example.pet_created]
  }

  screen "screen" {
    title = "Screen"
    actor = actor.user
  }

  screen_image "image" {
    title = "Image"
  }

  table "table" {
    title = "Table"
  }

  scenario "specification" {
    title = "Specification"

    when {
      command = command.command
    }

    then {
      event = event.example.pet_created
    }
  }
}

state_view "view_pets" {
  title = "View Pets"

  readmodel "readmodel" {
    title    = "Read model"
    question = "Which Read model are relevant?"
    from     = [event.example.pet_created]
  }
}

automation "notify_pet_created" {
  title = "Notify Pet Created"

  processor "processor" {
    title = "Processor"
    from  = [event.example.pet_created]
  }
}
