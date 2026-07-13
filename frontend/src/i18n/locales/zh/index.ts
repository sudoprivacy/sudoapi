import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import admin from './admin'
import misc from './misc'
import extension from './sudo.ts'
import { mergeLocaleMessages } from '../merge'

const messages = {
  ...landing,
  ...common,
  ...dashboard,
  admin,
  ...misc,
}

export default mergeLocaleMessages(messages, extension)
