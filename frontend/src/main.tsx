import { HashRouter } from 'react-router-dom';
import ReactDOM from 'react-dom/client';
import App from './App';
import './app/globals.css';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <HashRouter>
    <App />
  </HashRouter>
);
