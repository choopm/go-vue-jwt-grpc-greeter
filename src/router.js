import { createWebHistory, createRouter } from "vue-router";
import Login from "./components/Login.vue";
import Greet from "./components/Greet.vue";
// lazy-loaded
const Profile = () => import("./components/Profile.vue")

const routes = [
  {
    path: "/",
    name: "home",
    component: Greet,
  },
  {
    path: "/greet",
    component: Greet,
  },
  {
    path: "/login",
    component: Login,
  },
  {
    path: "/profile",
    name: "profile",
    // lazy-loaded
    component: Profile,
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach((to, from, next) => {
    const publicPages = ['/login'];
    const authRequired = !publicPages.includes(to.path);
    const loggedIn = localStorage.getItem('user');

    // trying to access a restricted page + not logged in
    // redirect to login page
    if (authRequired && !loggedIn) {
      next('/login');
    } else {
      next();
    }
});

export default router;
