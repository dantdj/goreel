import { useState, useEffect } from 'react'
import reactLogo from './assets/react.svg'
import viteLogo from '/vite.svg'
import './App.css'

function App() {
  const [count, setCount] = useState(0)
  const [ping, setPing] = useState<string>('Loading...')

  useEffect(() => {
    fetch('http://127.0.0.1:8089/ping')
      .then(async res => {
        if (!res.ok) {
          const text = await res.text();
          throw new Error(`Status: ${res.status} ${res.statusText} - ${text}`);
        }
        return res.json();
      })
      .then(data => setPing(JSON.stringify(data, null, 2)))
      .catch(err => setPing(`Error: ${err.message}`))
  }, [])

  return (
    <>
      <div>
        <a href="https://vite.dev" target="_blank">
          <img src={viteLogo} className="logo" alt="Vite logo" />
        </a>
        <a href="https://react.dev" target="_blank">
          <img src={reactLogo} className="logo react" alt="React logo" />
        </a>
      </div>
      <h1>Vite + React</h1>
      <div className="card">
        <h2>API Status</h2>
        <pre style={{ textAlign: 'left', background: '#f4f4f4', padding: '1rem', borderRadius: '8px', color: '#333' }}>
          {ping}
        </pre>
      </div>
      <div className="card">
        <button onClick={() => setCount((count) => count + 1)}>
          count is {count}
        </button>
        <p>
          Edit <code>src/App.tsx</code> and save to test HMR
        </p>
      </div>
      <p className="read-the-docs">
        Click on the Vite and React logos to learn more
      </p>
    </>
  )
}

export default App
