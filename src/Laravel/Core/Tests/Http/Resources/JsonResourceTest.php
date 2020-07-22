<?php

/*
 * This file is part of the Omed project.
 *
 * (c) Anthonius Munthi <https://itstoni.com>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

declare(strict_types=1);

namespace Omed\Laravel\Core\Http\Resources;

use Illuminate\Http\Request;
use Omed\Laravel\Core\Tests\TestCase;

class TestResource
{
    /**
     * @var string
     */
    private $id;

    /**
     * @var string
     */
    private $username;

    /**
     * @var string
     */
    private $password;

    /**
     * @return string
     */
    public function getId(): string
    {
        return $this->id;
    }

    /**
     * @param string $id
     */
    public function setId(string $id): void
    {
        $this->id = $id;
    }

    /**
     * @return string
     */
    public function getUsername(): string
    {
        return $this->username;
    }

    /**
     * @param string $username
     */
    public function setUsername(string $username): void
    {
        $this->username = $username;
    }

    /**
     * @return string
     */
    public function getPassword(): string
    {
        return $this->password;
    }

    /**
     * @param string $password
     */
    public function setPassword(string $password): void
    {
        $this->password = $password;
    }
}

class TestJsonResource extends JsonResource
{
    protected $filters = [
        'password',
    ];

    protected function getSelfRoute($resource)
    {
        return 'self-route';
    }
}

/**
 * Class JsonResourceTest.
 *
 * @covers \Omed\Laravel\Core\Http\Resources\JsonResource
 */
class JsonResourceTest extends TestCase
{
    public function testToArray()
    {
        $request = $this->createMock(Request::class);
        $resource = $this->createMock(TestResource::class);

        $resource->expects($this->once())
            ->method('getUsername')
            ->willReturn('username');

        $resource->expects($this->once())
            ->method('getId')
            ->willReturn('id');

        $ob = new TestJsonResource($resource);
        $data = $ob->toArray($request);

        $this->assertEquals('id', $data['id']);
        $this->assertEquals('username', $data['username']);
        $this->assertEquals('self-route', $data['_self']);
    }
}
