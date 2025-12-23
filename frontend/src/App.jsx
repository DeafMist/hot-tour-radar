import {useCallback, useRef, useState} from 'react';
import SearchBar from './components/SearchBar/SearchBar.jsx';
import Card from './components/Card/Card.jsx';

const API = `/api/news`;

export default function App() {
    const [results, setResults] = useState([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState(null);
    const controllerRef = useRef(null);

    const handleSearch = useCallback(async (query) => {
        if (controllerRef.current) {
            controllerRef.current.abort();
        }
        controllerRef.current = new AbortController();

        setLoading(true);
        setError(null);
        try {
            const res = query ?
                await fetch(`${API}?q=${encodeURIComponent(query)}`, { signal: controllerRef.current.signal }) :
                await fetch(API, { signal: controllerRef.current.signal });
            if (!res.ok) {
                throw new Error(`HTTP error! status: ${res.status}`);
            }
            const data = await res.json();
            setResults(Array.isArray(data.Items) ? data.Items : []);
        } catch (e) {
            if (e.name === 'AbortError') return;
            console.error(e);
            setError('Failed to load results. Please try again.');
        } finally {
            setLoading(false);
        }
    }, []);

    return (
        <div className="app-container">
            <div className="search-wrapper">
                <SearchBar onSearch={handleSearch}/>
            </div>

            <div className="cards-wrapper">
                {loading && <p>Loading...</p>}
                {error && <p className="error">{error}</p>}
                {!loading && results.length === 0 ?
                    <p>No news found :(</p>
                    :
                    results.map(item => (
                        <Card key={item.id} data={item}/>
                    ))}
            </div>
        </div>
    );
}
