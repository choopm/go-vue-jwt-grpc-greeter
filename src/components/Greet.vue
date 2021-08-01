<template>
  <div class="container">
    <div class="form-group">
      <label for="name">Name</label>
      <input name="name" type="text" class="form-control" placeholder="Your name" v-model="name"/>
    </div>
    <div class="form-group">
      <button class="btn btn-primary btn-block" :disabled="loading" @click="doGreet()">
        <span v-show="loading" class="spinner-border spinner-border-sm"></span>
        <span>Hello</span>
      </button>
    </div>

    <div class="form-group">
      <div v-if="greeting" class="alert alert-info" role="alert">
        {{ greeting }}
      </div>
    </div>
  </div>
</template>

<script>
import GreeterService from "../services/greeter.service";

export default {
  data() {
    return {
      loading: false,
      name: this.$store.state.auth.user.username,
      greeting: "",
    };
  },
  methods: {
    doGreet() {
      this.loading = true;
      GreeterService.hello(this.name).then(
        (response) => {
          this.loading = false;
          this.greeting = response.data.greeting;
        },
        (error) => {
          this.loading = false;
          this.greeting =
            (error.response &&
              error.response.data &&
              error.response.data.message) ||
            error.message ||
            error.toString();
        }
      );
    }
  },
};
</script>

<style scoped>
label {
  display: block;
  margin-top: 10px;
}
</style>
