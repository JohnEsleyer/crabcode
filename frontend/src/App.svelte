<script>
  import './style.css';
  import './app.css';
  import logo from './assets/images/logo-universal.png';
  import { Greet } from '../wailsjs/go/main/App';

  let name = '';
  let result = 'Please enter your name below 👇';

  function greet() {
    if (!name) return;
    Greet(name)
      .then((res) => { result = res; })
      .catch((err) => console.error(err));
  }
</script>

<img id="logo" class="logo" src={logo} alt="logo" />
<div class="result" id="result">{result}</div>
<div class="input-box" id="input">
  <input class="input" id="name" type="text" autocomplete="off" bind:value={name} />
  <button class="btn" onclick={greet}>Greet</button>
</div>
