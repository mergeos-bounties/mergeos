import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import './styles.css'

// Add font awesome icons if not already present
import { library } from '@fortawesome/fontawesome-svg-core'
import { faCode, faBug } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'

library.add(faCode, faBug)

const app = createApp(App)
app.component('font-awesome-icon', FontAwesomeIcon)
app.use(router)
app.mount('#app')