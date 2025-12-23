import React from 'react';
import ReactMarkdown from 'react-markdown';
import rehypeSanitize from 'rehype-sanitize';
import remarkBreaks from 'remark-breaks';
import './Card.css';

export default function Card({data}) {
    return (
        <div className="card">
            <ReactMarkdown
                remarkPlugins={[remarkBreaks]}
                rehypePlugins={[rehypeSanitize]}
                components={{
                    p: ({node, ...props}) => <h3 {...props} />,
                }}
            >
                {data.title}
            </ReactMarkdown>
            <ReactMarkdown
                remarkPlugins={[remarkBreaks]}
                rehypePlugins={[rehypeSanitize]}
            >
                {data.text}
            </ReactMarkdown>
        </div>
    );
}