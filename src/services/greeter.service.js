import axios from 'axios';
import authHeader from './auth-header';

const API_URL = '/api/hello';

class GreeterService {
  hello(username) {
    return axios.get(API_URL + "?name=" + username, { headers: authHeader() });
  }
}

export default new GreeterService();
