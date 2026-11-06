board {
  workflows = [
    workflow.open_booking,
    workflow.confirm_booking,
    workflow.cancel_booking,
    workflow.open_bookings,
    workflow.record_deposit,
  ]
}

actor "guest" {
  auth_required = false
}

bounded_context "booking" {
  field_type "booking_id" {
    type         = "UUID"
    id_attribute = true
  }

  field_type "guest_name" {
    type = "String"
  }

  field_type "payment_reference" {
    type = "String"
  }

  field_type "amount" {
    type = "Decimal"
  }

  aggregate "booking" {
    identity = field_type.booking_id
  }

  event "booking_opened" {
    aggregate = aggregate.booking
    fields    = [field_type.booking_id, field_type.guest_name]
  }

  event "booking_confirmed" {
    aggregate = aggregate.booking
    fields    = [field_type.booking_id]
  }

  event "booking_canceled" {
    aggregate = aggregate.booking
    fields    = [field_type.booking_id]
  }

  event "deposit_recorded" {
    aggregate = aggregate.booking
    fields    = [field_type.booking_id, field_type.payment_reference, field_type.amount]
  }
}

bounded_context "payments" {
  external = true

  event "payment_received" {
    field "booking_reference" {
      type            = "String"
      correlation_key = true
    }

    field "payment_reference" {
      type              = "String"
      deduplication_key = true
    }

    field "paid_amount" {
      type = "Decimal"
    }
  }
}

lifecycle "booking" {
  aggregate = aggregate.booking.booking

  state "open" {
    path = "main"
  }

  state "confirmed" {
    end  = true
    path = "main"
  }

  state "canceled" {
    end  = true
    path = "cancel"
  }

  transition "open_booking" {
    command = command.open_booking.open_booking
    events  = [event.booking.booking_opened]
    before  = []
    after   = state.open
  }

  transition "confirm_booking" {
    command = command.confirm_booking.confirm_booking
    events  = [event.booking.booking_confirmed]
    before  = [state.open]
    after   = state.confirmed
  }

  transition "cancel_booking" {
    command = command.cancel_booking.cancel_booking
    events  = [event.booking.booking_canceled]
    before  = [state.open]
    after   = state.canceled
  }

  invalid_transition "confirm_canceled_booking" {
    command = command.confirm_booking.confirm_booking
    reason  = "A canceled booking cannot be confirmed."
    before  = [state.canceled]
  }
}

state_change "open_booking" {
  screen "booking_form" {
    actor = actor.guest
    to    = [command.open_booking]

    field "name" {
      type = field_type.booking.guest_name
    }
  }

  command "open_booking" {
    fields = [field_type.booking.booking_id, field_type.booking.guest_name]
    to     = [event.booking.booking_opened]
  }

  field_source {
    inputs = [screen.booking_form.field.name]
    target = command.open_booking.field.guest_name
  }

  field_source {
    inputs = [command.open_booking.field.guest_name]
    target = event.booking.booking_opened.field.guest_name
  }

  scenario "booking_is_opened" {
    case = "happy_path"

    when {
      command = command.open_booking
    }

    then {
      event = event.booking.booking_opened
    }
  }
}

state_change "confirm_booking" {
  screen "booking_details" {
    actor = actor.guest
    to    = [command.confirm_booking]
  }

  command "confirm_booking" {
    fields = [field_type.booking.booking_id]
    to     = [event.booking.booking_confirmed]
  }

  scenario "open_booking_is_confirmed" {
    case = "happy_path"

    given {
      event = event.booking.booking_opened
    }

    when {
      command = command.confirm_booking
    }

    then {
      event = event.booking.booking_confirmed
    }
  }

  scenario "canceled_booking_is_not_confirmed" {
    case = "state_violation"

    given {
      event = event.booking.booking_opened
    }

    given {
      event = event.booking.booking_canceled
    }

    when {
      command = command.confirm_booking
    }

    then {
      error = "A canceled booking cannot be confirmed."
    }
  }
}

state_change "cancel_booking" {
  screen "booking_details" {
    actor = actor.guest
    to    = [command.cancel_booking]
  }

  command "cancel_booking" {
    fields = [field_type.booking.booking_id]
    to     = [event.booking.booking_canceled]
  }

  scenario "open_booking_is_canceled" {
    case = "happy_path"

    given {
      event = event.booking.booking_opened
    }

    when {
      command = command.cancel_booking
    }

    then {
      event = event.booking.booking_canceled
    }
  }
}

state_view "open_bookings" {
  readmodel "open_bookings" {
    question = "Which bookings are open?"
    fields   = [field_type.booking.booking_id, field_type.booking.guest_name]
    from     = [event.booking.booking_opened, event.booking.booking_canceled]
  }

  field_source {
    inputs = [event.booking.booking_opened.field.guest_name]
    target = readmodel.open_bookings.field.guest_name
  }

  scenario "no_booking_yet" {
    empty_history = true

    then {
      expect_empty_list = true
      readmodel         = readmodel.open_bookings
    }
  }

  scenario "opened_booking_appears" {
    case = "appear"

    given {
      event = event.booking.booking_opened
    }

    then {
      readmodel = readmodel.open_bookings
    }
  }

  scenario "canceled_booking_disappears" {
    case = "disappear"

    given {
      event = event.booking.booking_opened
    }

    given {
      event = event.booking.booking_canceled
    }

    then {
      expect_empty_list = true
      readmodel         = readmodel.open_bookings
    }
  }
}

automation "record_deposit" {
  processor "deposit_translator" {
    from = [event.payments.payment_received]
    to   = [command.record_deposit]

    field "payment_reference" {
      type = field_type.booking.payment_reference
    }
  }

  command "record_deposit" {
    fields = [field_type.booking.booking_id, field_type.booking.payment_reference, field_type.booking.amount]
    to     = [event.booking.deposit_recorded]
  }

  field_source {
    inputs = [event.payments.payment_received.field.payment_reference]
    target = processor.deposit_translator.field.payment_reference
  }

  scenario "payment_becomes_deposit" {
    case = "happy_path"

    given {
      event = event.payments.payment_received
    }

    when {
      processor = processor.deposit_translator
    }

    then {
      command = command.record_deposit
    }
  }

  scenario "repeated_payment_is_ignored" {
    case = "business_duplicate"

    given {
      event = event.booking.deposit_recorded
    }

    given {
      event = event.payments.payment_received
    }

    when {
      processor = processor.deposit_translator
    }

    then {
      expect_no_dispatch = true
      command            = command.record_deposit
    }
  }
}

evidence_source "booking_policy" {
  kind     = "document"
  location = "docs/booking-policy.pdf"
}

evidence "canceled_booking_rule" {
  certainty = "stated"
  location  = "section 2, page 3"
  source    = evidence_source.booking_policy
  on        = invalid_transition.booking.confirm_canceled_booking
}

evidence "payment_reference_is_unique" {
  description = "No source states that a payment reference is unique."
  certainty   = "assumed"
  on          = event.payments.payment_received.field.payment_reference
}

decision "one_open_bookings_list" {
  kind      = "accepted_debt"
  signal    = "EM403"
  statement = "The open bookings list reads two events."
  reason    = "The guest sees one list. A split gives two lists with one purpose."
  on        = readmodel.open_bookings.open_bookings
}

hotspot "refund_rule" {
  kind     = "assumption"
  category = "rule"
  question = "Does the guest lose the deposit after a cancellation?"
  options  = ["The guest loses the deposit.", "The guest gets the deposit back."]
  on       = command.cancel_booking.cancel_booking
}

hotspot "late_cancellation_scenario" {
  kind     = "request"
  question = "Which scenario states a cancellation on the day of the booking?"
  concerns = "scenario"
  on       = workflow.cancel_booking
}
