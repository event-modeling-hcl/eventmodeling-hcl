# Course subscriptions: canonical subscription contracts plus write/read workflows.

actor "student" {
  title         = "Student"
  auth_required = true
}

bounded_context "subscriptions" {
  title = "Subscriptions"

  aggregate "subscription" {
  }

  field_type "student_id" {
    type         = "UUID"
    id_attribute = true
    example      = "a3f1c2d4-0000-4000-8000-000000000001"
  }

  field_type "course_id" {
    type    = "UUID"
    example = "b7e9d1a2-0000-4000-8000-000000000002"
  }

  field_type "subscribed_at" {
    type    = "DateTime"
    example = "2026-09-04T09:00:00Z"
  }

  event "student_subscribed" {
    title     = "Student Subscribed"
    aggregate = aggregate.subscription

    field "student_id" {
      type = field_type.student_id
    }

    field "course_id" {
      type = field_type.course_id
    }

    field "subscribed_at" {
      type = field_type.subscribed_at
    }
  }
}

state_change "subscribe_to_course" {
  title = "Subscribe to Course"

  screen "subscription_form" {
    title = "Subscription Form"
    actor = actor.student
    to    = [command.subscribe_to_course_command]
  }

  command "subscribe_to_course_command" {
    title        = "Subscribe to Course"
    aggregate    = aggregate.subscriptions.subscription
    api_endpoint = "POST /courses/{courseId}/subscriptions"
    to           = [event.subscriptions.student_subscribed]

    field "student_id" {
      type = field_type.subscriptions.student_id
    }

    field "course_id" {
      type = field_type.subscriptions.course_id
    }

    field "seats" {
      type    = "Int"
      example = 1
    }
  }

  scenario "subscribe_successfully" {
    title = "Subscribe a student to a course"

    when {
      title   = "Subscribe to Course"
      command = command.subscribe_to_course_command

      field "student_id" {
        type = field_type.subscriptions.student_id
      }

      field "course_id" {
        type = field_type.subscriptions.course_id
      }
    }

    then {
      title = "Student Subscribed"
      event = event.subscriptions.student_subscribed

      field "student_id" {
        type = field_type.subscriptions.student_id
      }
    }
  }
}

state_view "view_subscriptions" {
  title = "View Subscriptions"

  readmodel "student_subscriptions" {
    title    = "Student Subscriptions"
    question = "Which courses is the student subscribed to?"
    from     = [event.subscriptions.student_subscribed]
    to       = [screen.subscriptions_list]

    field "student_id" {
      type = field_type.subscriptions.student_id
    }

    field "courses" {
      cardinality = "List"
      type        = "Custom"
      example = [{
        course_id = "b7e9d1a2-0000-4000-8000-000000000002"
        title     = "Event Modeling 101"
      }]

      subfield "course_id" {
        type    = "UUID"
        example = "b7e9d1a2-0000-4000-8000-000000000002"
      }

      subfield "title" {
        type    = "String"
        example = "Event Modeling 101"
      }
    }
  }

  screen "subscriptions_list" {
    title = "Subscriptions List"
    actor = actor.student
  }

  scenario "list_active_subscriptions" {
    title = "List a student's active subscriptions"

    given {
      title = "Student Subscribed"
      event = event.subscriptions.student_subscribed

      field "student_id" {
        type = field_type.subscriptions.student_id
      }

      field "course_id" {
        type = field_type.subscriptions.course_id
      }
    }

    then {
      title     = "Student Subscriptions"
      readmodel = readmodel.student_subscriptions

      field "courses" {
        cardinality = "List"
        type        = "Custom"
        example = [{
          course_id = "b7e9d1a2-0000-4000-8000-000000000002"
          title     = "Event Modeling 101"
        }]
      }
    }
  }
}
